package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

//go:embed assets/*.png assets/fonts/*.ttf assets/audio/*.mp3
var assets embed.FS
var statePath string

type Settings struct {
	Schema                           int
	Journey                          Journey
	Language, Typography, Appearance string
	Cruise                           bool
	Audio                            AudioSettings
}
type Game struct {
	Settings
	drive                 DriveRuntime
	radio                 *Radio
	window                *mygo.Window
	benchmarking, reduced bool
	status, audioError    string
}

func (a *Game) tr(en, zh, tw string) string {
	switch a.Language {
	case "简体中文":
		return zh
	case "繁體中文":
		return tw
	}
	return en
}
func (a *Game) save() {
	if a.radio != nil {
		a.Audio = a.radio.settings()
	}
	if err := saveSettings(statePath, a.Settings); err != nil {
		a.status = a.tr("Save failed: ", "保存失败：", "儲存失敗：") + err.Error()
	} else {
		a.status = ""
	}
}
func (a *Game) pause(paused bool) {
	a.drive.paused = paused
	a.drive.clearInput()
	if a.radio != nil {
		a.radio.activeGame(!paused && !a.drive.maps && !a.drive.options && !a.reduced)
	}
	a.save()
}
func (a *Game) audioAction(f func() error) {
	if err := f(); err != nil {
		a.audioError = err.Error()
	}
	a.save()
	a.drive.focus = true
}
func (a *Game) fontLabel() string {
	if a.Typography == "Pixel" {
		return a.tr("Pixel", "像素", "像素")
	}
	return a.tr("System", "系统", "系統")
}

// The game rasterizes its original artwork on one deliberate coarse grid at
// load time. Fourfold nearest-neighbor expansion keeps the blocks crisp through
// the GPU's image filtering; no images are allocated during a frame.
func pixelBitmap(src image.Image, region image.Rectangle, columns int) *ui.Bitmap {
	h := max(1, region.Dy()*columns/region.Dx())
	out := image.NewNRGBA(image.Rect(0, 0, columns*4, h*4))
	for y := 0; y < h; y++ {
		for x := 0; x < columns; x++ {
			c := color.NRGBAModel.Convert(src.At(region.Min.X+x*region.Dx()/columns, region.Min.Y+y*region.Dy()/h)).(color.NRGBA)
			for yy := 0; yy < 4; yy++ {
				for xx := 0; xx < 4; xx++ {
					out.SetNRGBA(x*4+xx, y*4+yy, c)
				}
			}
		}
	}
	return ui.NewBitmap(out)
}
func decodeAsset(name string) image.Image {
	b, e := assets.ReadFile("assets/" + name + ".png")
	if e != nil {
		log.Fatal(e)
	}
	im, _, e := image.Decode(bytes.NewReader(b))
	if e != nil {
		log.Fatal(e)
	}
	return im
}
func trimSprite(src image.Image, cell image.Rectangle) image.Rectangle {
	result := image.Rectangle{}
	first := true
	for y := cell.Min.Y; y < cell.Max.Y; y++ {
		for x := cell.Min.X; x < cell.Max.X; x++ {
			_, _, _, alpha := src.At(x, y).RGBA()
			if alpha > 32767 {
				p := image.Rect(x, y, x+1, y+1)
				if first {
					result = p
					first = false
				} else {
					result = result.Union(p)
				}
			}
		}
	}
	if first {
		return cell
	}
	return result
}
func (a *Game) loadArt() {
	horizon := decodeAsset("horizon-sheet")
	hb := horizon.Bounds()
	for i := 0; i < 4; i++ {
		a.drive.skies[i] = pixelBitmap(horizon, image.Rect(0, i*hb.Dy()/4, hb.Dx(), (i+1)*hb.Dy()/4), 384)
	}
	sheet := decodeAsset("hatchback-sheet")
	bounds := sheet.Bounds()
	cw, ch := bounds.Dx()/3, bounds.Dy()/2
	for i := 0; i < 6; i++ {
		cell := image.Rect(i%3*cw, i/3*ch, (i%3+1)*cw, (i/3+1)*ch)
		trim := trimSprite(sheet, cell)
		a.drive.cars[i] = pixelBitmap(sheet, trim, 96)
	}
	props := decodeAsset("roadside-sheet")
	pb := props.Bounds()
	pw, ph := pb.Dx()/4, pb.Dy()/4
	for i := 0; i < 16; i++ {
		cell := image.Rect(i%4*pw, i/4*ph, (i%4+1)*pw, (i/4+1)*ph)
		a.drive.props[i] = pixelBitmap(props, trimSprite(props, cell), 72)
	}
	traffic := decodeAsset("car")
	a.drive.trafficCar = pixelBitmap(traffic, traffic.Bounds(), 80)
	a.drive.initTraffic(a.Journey.Seed + a.Journey.Region)
}
func main() {
	profile := flag.String("profile", "", "Write bounded-run CPU profile for rendering diagnosis")
	capture := flag.String("capture", "", "Save a native window PNG after a bounded run")
	benchmark := flag.Duration("benchmark", 0, "Bounded visible run and frame report; uses the specified state path")
	width := flag.Int("width", 1180, "Initial window width (minimum 980)")
	height := flag.Int("height", 780, "Initial window height (minimum 680)")
	preview := flag.String("preview", "", "Verification overlay: places or options")
	muteOutput := flag.Bool("mute-output", false, "Exercise audio transport at zero output gain")
	silent := flag.Bool("silent", false, "Disable audio output for isolated visual verification")
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&statePath, "state", filepath.Join(configDir, "Moonbounce", "settings.json"), "Local journey/settings file")
	flag.Parse()
	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			log.Fatal(err)
		}
		if err = pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		defer pprof.StopCPUProfile()
	}
	defaults := Settings{Schema: 2, Journey: newJourney(20261008), Language: "English", Typography: "Pixel", Appearance: "System", Cruise: false, Audio: AudioSettings{Track: 0, Music: true, Effects: true, Volume: .55}}
	saved, warning, err := loadSettings(statePath, defaults)
	if err != nil {
		log.Fatal(err)
	}
	a := &Game{Settings: saved, benchmarking: *benchmark > 0, status: warning}
	a.drive.race.Nitro = .6
	a.drive.paused = true
	a.drive.focus = true
	a.drive.maps = *preview == "places"
	a.drive.options = *preview == "options"
	a.loadArt()
	mygo.App.WhenReady(func() {
		font, e := assets.ReadFile("assets/fonts/DeparturePixelZhCompact-Regular.ttf")
		if e != nil {
			log.Fatal(e)
		}
		if e = ui.RegisterFont(font, "Moonbounce Pixel"); e != nil {
			log.Fatal(e)
		}
		a.window = mygo.NewWindow(mygo.WindowOptions{Title: "Moonbounce", Width: max(980, *width), Height: max(680, *height), MinWidth: 980, MinHeight: 680, Content: ui.View(a.view)})
		win := a.window
		if !*silent {
			a.radio, e = newRadio(a.Audio, *muteOutput)
			if e != nil {
				a.audioError = e.Error()
			}
		}
		win.OnBlur(func() {
			a.drive.clearInput()
			if !a.benchmarking {
				a.pause(true)
				win.Invalidate()
			}
		})
		win.OnClose(func(e *mygo.CloseEvent) { a.pause(true); a.radio.close() })
		if *benchmark > 0 {
			a.drive.paused = false
			time.AfterFunc(*benchmark, func() {
				win.Update(func() {
					if *capture != "" {
						b, e := win.CapturePage()
						if e != nil {
							log.Print(e)
						} else if e = os.WriteFile(*capture, b, 0600); e != nil {
							log.Print(e)
						}
					}
					samples := append([]float64(nil), a.drive.paintIntervals...)
					sort.Float64s(samples)
					sum := 0.0
					for _, s := range samples {
						sum += s
					}
					report := map[string]any{"version": "0.4.1", "framework": "MyGo v0.3.7", "paintedFrames": a.drive.painted, "sampleCount": len(samples), "reduceMotion": a.reduced, "width": *width, "height": *height, "language": a.Language, "typography": a.Typography, "theme": a.Appearance, "audioEnabled": a.radio != nil, "audioOutputMuted": *muteOutput, "sprintMeters": a.drive.race.Meters}
					if len(samples) > 0 {
						report["meanFPS"] = 1000 / (sum / float64(len(samples)))
						report["meanFrameMs"] = sum / float64(len(samples))
						report["p95FrameMs"] = samples[int(float64(len(samples)-1)*.95)]
					}
					out, _ := json.MarshalIndent(report, "", "  ")
					fmt.Println(string(out))
					win.Close()
				})
			})
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
