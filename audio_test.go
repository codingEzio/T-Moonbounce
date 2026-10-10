package main

import (
	"encoding/binary"
	"io"
	"math"
	"os"
	"testing"
	"time"
)

func TestBundledTracksDecodeSeekAndLoop(t *testing.T) {
	for i, track := range tracks {
		t.Run(track.Title, func(t *testing.T) {
			s, err := newSong(i)
			if err != nil {
				t.Fatal(err)
			}
			if s.decoder.Length() < audioRate*4*60 {
				t.Fatal("track unexpectedly short")
			}
			buf := make([]byte, audioRate*4)
			if _, err = s.Seek(audioRate*4*12, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			n, err := io.ReadFull(s, buf)
			if err != nil || n != len(buf) {
				t.Fatalf("decode: %d %v", n, err)
			}
			peak := 0.0
			sum := 0.0
			for p := 0; p < n; p += 2 {
				v := float64(int16(binary.LittleEndian.Uint16(buf[p:]))) / 32768
				peak = math.Max(peak, math.Abs(v))
				sum += v * v
			}
			rms := math.Sqrt(sum / float64(n/2))
			if rms < .002 || peak > 1 {
				t.Fatalf("invalid audio signal RMS %.5f peak %.5f", rms, peak)
			}
			if _, err = s.Seek(s.decoder.Length()-4000, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			if _, err = io.ReadFull(s, buf[:12000]); err != nil {
				t.Fatal(err)
			}
			_, loops := s.state()
			if loops != 1 {
				t.Fatalf("EOF did not loop once: %d", loops)
			}
			t.Logf("48 kHz stereo, %.1f sec, RMS %.4f, peak %.4f, seamless read across EOF", float64(s.decoder.Length())/(audioRate*4), rms, peak)
		})
	}
}
func TestEngineSignalTracksSpeedWithoutClipping(t *testing.T) {
	e := &engineStream{rng: 42}
	p := make([]byte, audioRate*4)
	e.set(70, 1, 2, 1)
	if n, err := e.Read(p); err != nil || n != len(p) {
		t.Fatalf("%d %v", n, err)
	}
	sum := 0.0
	peak := 0.0
	for i := 0; i < len(p); i += 2 {
		v := float64(int16(binary.LittleEndian.Uint16(p[i:]))) / 32768
		sum += v * v
		peak = math.Max(peak, math.Abs(v))
	}
	if sum < 1 || peak > .4 {
		t.Fatalf("engine silent or clipping: energy %.2f peak %.3f", sum, peak)
	}
	if e.count() != audioRate {
		t.Fatal("sample clock mismatch")
	}
}

// Opt in to actual CoreAudio output. This is a bounded audio-device check,
// not a UI test and not included in an ordinary `go test` run.
func TestNativeAudioTransport(t *testing.T) {
	if os.Getenv("MOONBOUNCE_AUDIO_TEST") != "1" {
		t.Skip("set MOONBOUNCE_AUDIO_TEST=1 for actual local audio output")
	}
	r, err := newRadio(AudioSettings{Track: 0, Music: true, Effects: true, Volume: .12}, os.Getenv("MOONBOUNCE_MUTE_OUTPUT") == "1")
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()
	if r.muted {
		r.setVolume(1)
		if r.music.Volume() != 0 || r.engine.Volume() != 0 {
			t.Fatal("test mute allowed audible gain")
		}
	}
	musicPlayer := r.music
	enginePlayer := r.engine
	wait := func(what string, predicate func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if predicate() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("timed out: " + what)
	}
	r.activeGame(true)
	wait("music reads", func() bool { p, _ := r.source.state(); return p > 0 && r.music.IsPlaying() })
	r.activeGame(false)
	p, _ := r.source.state()
	frames := r.motor.count()
	time.Sleep(100 * time.Millisecond)
	p2, _ := r.source.state()
	if p != p2 || frames != r.motor.count() || r.music.IsPlaying() || r.engine.IsPlaying() {
		t.Fatal("pause did not freeze both sources")
	}
	r.activeGame(true)
	wait("resume preserves and advances", func() bool { p2, _ := r.source.state(); return p2 > p })
	if err = r.stop(); err != nil {
		t.Fatal(err)
	}
	if r.music.IsPlaying() {
		t.Fatal("stop still plays")
	}
	p, _ = r.source.state()
	if p != 0 {
		t.Fatalf("stop position %d", p)
	}
	r.activeGame(false)
	r.activeGame(true)
	if r.music.IsPlaying() {
		t.Fatal("manual stop was lost on game resume")
	}
	if err = r.replay(); err != nil {
		t.Fatal(err)
	}
	wait("replay", func() bool { return r.music.IsPlaying() })
	if err = r.next(); err != nil {
		t.Fatal(err)
	}
	if r.settings().Track != 1 {
		t.Fatal("next track not applied")
	}
	if r.music != musicPlayer || r.engine != enginePlayer {
		t.Fatal("transport allocated duplicate players")
	}
	for i := 0; i < 20; i++ {
		r.toggle()
	}
	if !r.music.IsPlaying() {
		t.Fatal("rapid even toggles changed play intent")
	}
	r.activeGame(false)
	playing, _, err := r.status()
	if err != nil || playing {
		t.Fatalf("final pause: %v %v", playing, err)
	}
	t.Log("actual Oto/CoreAudio: play, freeze, resume, stop, replay, next, 20 toggles; same two player identities throughout")
}

// A read crossing the MP3 loop boundary must sound the same regardless of the
// output device's buffer size. Guard the per-segment attack and release.
func TestLoopEnvelopeIndependentOfReadSize(t *testing.T) {
	a, err := newSong(0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := newSong(0)
	if err != nil {
		t.Fatal(err)
	}
	end := a.decoder.Length()
	if _, err := a.Seek(end-4000, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Seek(end-4000, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	whole := make([]byte, 12000)
	parts := make([]byte, 12000)
	if _, err := io.ReadFull(a, whole); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(b, parts[:4000]); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(b, parts[4000:]); err != nil {
		t.Fatal(err)
	}
	for i := range whole {
		if whole[i] != parts[i] {
			t.Fatalf("loop envelope changes with device buffer at byte %d", i)
		}
	}
}
