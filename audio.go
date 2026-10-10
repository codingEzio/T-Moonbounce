package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

const audioRate = 48000

var tracks = []struct{ File, Title string }{
	{"windows-down.mp3", "Windows Down"},
	{"keeping-cool.mp3", "Keeping Cool"},
	{"first-snow.mp3", "First Snow"},
}

type AudioSettings struct {
	Track                   int
	Music, Effects, Stopped bool
	Volume                  float64
	PositionBytes           int64
}

// One context and exactly one player per purpose. Transport operations are
// serialized; a paused source stops reading, and seeks flush Oto's queued PCM.
type Radio struct {
	mu                       sync.Mutex
	context                  *oto.Context
	music, engine            *oto.Player
	source                   *songStream
	motor                    *engineStream
	enabled, effects, active bool
	stopped                  bool
	closed                   bool
	muted                    bool
	track                    int
	volume                   float64
}

type songStream struct {
	mu       sync.Mutex
	decoder  *mp3.Decoder
	position int64
	loops    uint64
	ramp     int
}

func newSong(index int) (*songStream, error) {
	data, err := assets.ReadFile("assets/audio/" + tracks[index].File)
	if err != nil {
		return nil, err
	}
	dec, err := mp3.NewDecoder(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if dec.SampleRate() != audioRate {
		return nil, fmt.Errorf("track sample rate %d; expected %d", dec.SampleRate(), audioRate)
	}
	return &songStream{decoder: dec}, nil
}
func (s *songStream) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := 0
	for total < len(p) {
		n, err := s.decoder.Read(p[total:])
		s.shape(p[total : total+n])
		s.position += int64(n)
		total += n
		if err == io.EOF {
			if _, e := s.decoder.Seek(0, io.SeekStart); e != nil {
				return total, e
			}
			s.position = 0
			s.loops++
			s.ramp = 0
			if n == 0 && total == 0 {
				n2, e := s.decoder.Read(p)
				if e != nil {
					return n2, e
				}
				s.shape(p[:n2])
				s.position += int64(n2)
				total = n2
			}
			continue
		}
		if err != nil {
			return total, err
		}
		if n == 0 {
			return total, io.ErrNoProgress
		}
	}
	return total, nil
}

// Apply the envelope to each decoded segment before changing the loop position.
// This remains consistent when a CoreAudio read crosses the end of a track.
func (s *songStream) shape(p []byte) {
	const fadeFrames = audioRate / 50
	for off := 0; off+3 < len(p); off += 4 {
		gain := math.Min(1, float64(s.ramp)/fadeFrames)
		remaining := (s.decoder.Length() - s.position - int64(off)) / 4
		gain = math.Min(gain, math.Max(0, float64(remaining)/fadeFrames))
		for ch := 0; ch < 2; ch++ {
			i := off + ch*2
			v := int16(binary.LittleEndian.Uint16(p[i:]))
			binary.LittleEndian.PutUint16(p[i:], uint16(int16(float64(v)*gain)))
		}
		if s.ramp < fadeFrames {
			s.ramp++
		}
	}
}

func (s *songStream) Seek(offset int64, whence int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, e := s.decoder.Seek(offset, whence)
	if e == nil {
		s.position = n
		s.ramp = 0
	}
	return n, e
}
func (s *songStream) state() (int64, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.position, s.loops
}

// A quiet continuous engine, tire and wind bed responds to actual simulation
// speed and slip. Phase survives pauses; no per-frame sample/player allocation.
type engineStream struct {
	mu                                                          sync.Mutex
	speed, slip, boost, throttle, level, phase, noise, filtered float64
	rng                                                         uint64
	frames                                                      uint64
}

func (e *engineStream) set(speed, slip, boost, throttle float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.speed = speed
	e.slip = math.Abs(slip)
	e.boost = boost
	e.throttle = throttle
}
func (e *engineStream) Read(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := len(p) / 4 * 4
	gear := math.Min(5, math.Floor(e.speed/13))
	rpm := 38 + (e.speed-gear*13)*2.5 + e.throttle*12
	if e.boost > 0 {
		rpm += 10
	}
	target := .045 + math.Min(e.speed/70, 1)*.055
	for i := 0; i < n; i += 4 {
		e.phase += rpm / audioRate
		e.phase -= math.Floor(e.phase)
		e.rng = e.rng*6364136223846793005 + 1442695040888963407
		noise := float64(e.rng>>32)/float64(uint64(1)<<32)*2 - 1
		e.filtered += .035 * (noise - e.filtered)
		e.level += (target - e.level) * .0008
		engine := (math.Sin(e.phase*2*math.Pi) + .25*math.Sin(e.phase*4*math.Pi)) * .48
		wind := e.filtered * math.Min(e.speed/100, .8)
		tires := noise * math.Min(e.slip*.035, .06)
		v := (engine + wind + tires) * e.level
		sample := int16(math.Max(-.4, math.Min(.4, v)) * 32767)
		binary.LittleEndian.PutUint16(p[i:], uint16(sample))
		binary.LittleEndian.PutUint16(p[i+2:], uint16(sample))
		e.frames++
	}
	return n, nil
}
func (e *engineStream) count() uint64 { e.mu.Lock(); defer e.mu.Unlock(); return e.frames }

func newRadio(s AudioSettings, muteOutput ...bool) (*Radio, error) {
	song, err := newSong(s.Track)
	if err != nil {
		return nil, err
	}
	if s.PositionBytes > 0 && !s.Stopped {
		if _, err = song.Seek(s.PositionBytes%song.decoder.Length()/4*4, io.SeekStart); err != nil {
			return nil, err
		}
	}
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{SampleRate: audioRate, ChannelCount: 2, Format: oto.FormatSignedInt16LE, BufferSize: 35 * time.Millisecond, ApplicationName: "Moonbounce"})
	if err != nil {
		return nil, err
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("audio device initialization timed out")
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	motor := &engineStream{rng: 7}
	r := &Radio{context: ctx, source: song, motor: motor, enabled: s.Music, effects: s.Effects, track: s.Track, volume: s.Volume, stopped: s.Stopped}
	r.music = ctx.NewPlayer(song)
	r.engine = ctx.NewPlayer(motor)
	// 40 ms queue bounds transport latency while leaving headroom for decoding.
	r.music.SetBufferSize(audioRate * 4 / 25)
	r.engine.SetBufferSize(audioRate * 4 / 25)
	r.muted = len(muteOutput) > 0 && muteOutput[0]
	r.music.SetVolume(s.Volume * .65)
	r.engine.SetVolume(.65)
	if r.muted {
		r.music.SetVolume(0)
		r.engine.SetVolume(0)
	}
	r.music.PauseAndStopReading()
	r.engine.PauseAndStopReading()
	return r, nil
}
func (r *Radio) syncLocked() {
	if r.closed {
		return
	}
	if r.active && r.enabled {
		r.music.Play()
	} else {
		r.music.PauseAndStopReading()
	}
	if r.active && r.effects {
		r.engine.Play()
	} else {
		r.engine.PauseAndStopReading()
	}
}
func (r *Radio) activeGame(active bool) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != active {
		r.active = active
		r.syncLocked()
	}
}
func (r *Radio) toggle() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = !r.enabled
	r.stopped = false
	r.syncLocked()
}
func (r *Radio) stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = false
	r.stopped = true
	r.music.PauseAndStopReading()
	_, err := r.music.Seek(0, io.SeekStart)
	return err
}
func (r *Radio) replay() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.music.PauseAndStopReading()
	_, err := r.music.Seek(0, io.SeekStart)
	if err == nil {
		r.enabled = true
		r.stopped = false
		r.syncLocked()
	}
	return err
}
func (r *Radio) next() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	idx := (r.track + 1) % len(tracks)
	next, err := newSong(idx)
	if err != nil {
		return err
	}
	r.music.PauseAndStopReading()
	// Keep the existing player: changing the decoder then seeking flushes its buffer.
	r.source.mu.Lock()
	r.source.decoder = next.decoder
	r.source.position = 0
	r.source.loops = 0
	r.source.ramp = 0
	r.source.mu.Unlock()
	if _, err = r.music.Seek(0, io.SeekStart); err != nil {
		return err
	}
	r.track = idx
	r.enabled = true
	r.stopped = false
	r.syncLocked()
	return nil
}
func (r *Radio) toggleEffects() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.effects = !r.effects
	r.syncLocked()
}
func (r *Radio) setVolume(v float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.volume = math.Max(0, math.Min(1, v))
	if !r.muted {
		r.music.SetVolume(r.volume * .65)
	}
}
func (r *Radio) settings() AudioSettings {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, _ := r.source.state()
	p = max(0, p-int64(r.music.BufferedSize()))
	return AudioSettings{Track: r.track, Music: r.enabled, Effects: r.effects, Stopped: r.stopped, Volume: r.volume, PositionBytes: p}
}
func (r *Radio) status() (bool, float64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, _ := r.source.state()
	if err := r.context.Err(); err != nil {
		return false, 0, err
	}
	return r.music.IsPlaying(), float64(p-int64(r.music.BufferedSize())) / float64(audioRate*4), r.music.Err()
}
func (r *Radio) close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active = false
	r.syncLocked()
	r.closed = true
}
