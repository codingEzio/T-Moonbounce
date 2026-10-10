package main

import (
	"fmt"
	"math"
	"math/bits"
	"time"

	"github.com/egoist/mygo/ui"
)

type Traffic struct {
	Z, X, Speed float64
	ID          uint64
}
type DriveRuntime struct {
	race                                                                  Sprint
	drafting                                                              bool
	props                                                                 [16]*ui.Bitmap
	audioChecked                                                          time.Time
	musicSeconds                                                          float64
	skies                                                                 [4]*ui.Bitmap
	cars                                                                  [6]*ui.Bitmap
	trafficCar                                                            *ui.Bitmap
	left, right, brake, gas, drift, pointer, paused, maps, options, focus bool
	pointerTarget                                                         float64
	last, lastSave, lastPaint, timeStart                                  time.Time
	clock, eventUntil, fps                                                float64
	event                                                                 int
	painted, frames                                                       int
	paintIntervals                                                        []float64
	stageWidth                                                            float64
	traffic                                                               [6]Traffic
	roadX                                                                 [161]float64
}

func rgb(v uint32) ui.Color { return ui.RGB(uint8(v>>16), uint8(v>>8), uint8(v)) }
func (d *DriveRuntime) clearInput() {
	d.left = false
	d.right = false
	d.brake = false
	d.gas = false
	d.drift = false
	d.pointer = false
}
func (d *DriveRuntime) initTraffic(seed uint64) {
	for i := range d.traffic {
		d.traffic[i] = Traffic{24 + float64(i)*25, []float64{-.6, 0, .6}[i%3], 18 + random01(seed+uint64(i))*10, seed + uint64(i)}
	}
}
func (a *Game) driveInput(e ui.InputEvent) bool {
	d := &a.drive
	if e.Kind == ui.InputKeyDown || e.Kind == ui.InputKeyUp {
		if e.Mods&ui.Cmd != 0 {
			return false
		}
		down := e.Kind == ui.InputKeyDown
		switch e.Key {
		case ui.KeyLeft, ui.KeyA:
			d.left = down
		case ui.KeyRight, ui.KeyD:
			d.right = down
		case ui.KeyUp, ui.KeyW:
			d.gas = down
		case ui.KeyDown, ui.KeyS:
			d.brake = down
		case ui.KeySpace:
			if down && !e.Repeat && !d.paused {
				if d.race.Nitro >= .30 && a.Journey.Boost <= 0 {
					a.Journey.Boost = 1.8
					d.race.Nitro -= .30
				}
			}
			return true
		case ui.KeyP, ui.KeyEscape:
			if down && !e.Repeat {
				if d.maps || d.options {
					d.maps = false
					d.options = false
				} else {
					a.pause(!d.paused)
				}
				d.focus = true
			}
			return true
		case ui.KeyC:
			if down && !e.Repeat {
				a.Cruise = !a.Cruise
				a.save()
			}
			return true
		default:
			return false
		}
		return true
	}
	switch e.Kind {
	case ui.InputPointerDown:
		if e.Button == 0 {
			d.pointer = true
			d.pointerTarget = math.Max(-1, math.Min(1, (float64(e.X)/math.Max(d.stageWidth, 1)-.5)*2))
			return true
		}
	case ui.InputPointerMove:
		if d.pointer {
			d.pointerTarget = math.Max(-1, math.Min(1, (float64(e.X)/math.Max(d.stageWidth, 1)-.5)*2))
			return true
		}
	case ui.InputPointerUp:
		if d.pointer {
			d.pointer = false
			return true
		}
	}
	return false
}
func gameButton(c *ui.Context, key, label string, active bool) ui.Element {
	t := c.Theme()
	b := ui.Button(c.Key(key), label).Radius(0).Border(2, t.Border).Shadow(0, 3, 0, 0, ui.Hex("#151d32")).Padding(7, 12).FontSize(12)
	if active {
		b.Background(t.Accent).TextColor(t.AccentText)
	}
	return b
}
func (a *Game) update(c *ui.Context) {
	a.reduced = c.Preferences().ReduceMotion
	if a.reduced {
		a.drive.paused = true
	}
	a.drive.drift = c.Modifiers()&ui.Shift != 0
	a.advance(c.Now())
}

// Simulation and scene painting follow the display clock. Numeric HUD labels
// refresh separately, avoiding full control layout on every Metal frame.
func (a *Game) advance(now time.Time) {
	d, j := &a.drive, &a.Journey
	active := !d.paused && !d.maps && !d.options && !a.reduced
	if a.radio != nil {
		a.radio.activeGame(active)
		if now.Sub(d.audioChecked) > time.Second {
			_, seconds, err := a.radio.status()
			d.musicSeconds = seconds
			if err != nil {
				a.audioError = err.Error()
			}
			d.audioChecked = now
		}
	}
	if now.Equal(d.last) {
		return
	}
	dt := now.Sub(d.last).Seconds()
	d.last = now
	if dt > .2 {
		dt = 0
		d.timeStart = now
		d.frames = 0
	}
	if !active {
		return
	}
	steer := 0.0
	if d.left {
		steer--
	}
	if d.right {
		steer++
	}
	if d.pointer {
		steer = d.pointerTarget
	}
	throttle := 0.0
	if d.gas || a.benchmarking {
		throttle = 1
	}
	before := j.Region
	beforeKM := j.Kilometers
	j.drive(dt, Controls{steer, throttle, d.brake, d.drift, a.Cruise})
	d.clock += math.Min(dt, .05)
	if before != j.Region {
		d.event = 3
		d.eventUntil = d.clock + 3
	}
	d.drafting = false
	for i := range d.traffic {
		car := &d.traffic[i]
		old := car.Z
		if i < 3 {
			// Rivals stay on a predictable racing line. Overtaking is the player's job.
			car.Speed = feels[j.Feel].Speed * (.90 + float64(i)*.065)
			car.Z += (car.Speed - j.Speed) * math.Min(dt, .05)
			if car.Z > 6 && car.Z < 65 && math.Abs(car.X-j.X) < .24 {
				d.drafting = true
			}
		} else {
			car.Z += (car.Speed - j.Speed) * math.Min(dt, .05)
			if car.Z < -25 || car.Z > 1000 {
				car.Z = 500 + float64(i)*45
				car.X = []float64{-.65, 0, .65}[car.ID%3]
				car.ID++
			}
		}
		if old >= 3.8 && car.Z < 3.8 {
			j.Passes++
			d.event = 1
			d.eventUntil = d.clock + 1.4
			d.race.Nitro = math.Min(1, d.race.Nitro+.12)
		}
		// A brush costs momentum, never accumulated progress or the whole run.
		if car.Z > 2.8 && car.Z < 5.8 && math.Abs(car.X-j.X) < .22 {
			j.Speed *= math.Exp(-math.Min(dt, .05) * 1.7)
			j.X += math.Copysign(math.Min(dt, .05)*.9, j.X-car.X+.001)
		}
	}
	d.race.advance((j.Kilometers-beforeKM)*1000, math.Min(dt, .05), d.drafting, d.drift && math.Abs(j.Lateral) > .2)
	if d.race.Meters >= sprintLength {
		d.race.LastPlace = d.place()
		d.race.Number++
		d.race.Meters -= sprintLength
		d.event = 4
		d.eventUntil = d.clock + 4
		for i := 0; i < 3; i++ {
			d.traffic[i].Z = 24 + float64(i)*25
			d.traffic[i].X = []float64{-.6, 0, .6}[i]
		}
	}
	if a.radio != nil {
		slip := 0.0
		if d.drift {
			slip = j.Lateral
		}
		a.radio.motor.set(j.Speed, slip, j.Boost, throttle)
	}
	d.frames++
	if d.timeStart.IsZero() {
		d.timeStart = now
	}
	if elapsed := now.Sub(d.timeStart).Seconds(); elapsed >= .8 {
		d.fps = float64(d.frames) / elapsed
		d.frames = 0
		d.timeStart = now
	}
	if d.lastSave.IsZero() {
		d.lastSave = now
	}
	if now.Sub(d.lastSave) > 10*time.Second {
		if !a.benchmarking {
			a.save()
		}
		d.lastSave = now
	}
}
func (a *Game) view(c *ui.Context) {
	d := &a.drive
	t := *c.Theme()
	if a.Appearance == "Light" {
		t = *ui.LightTheme()
	} else if a.Appearance == "Dark" {
		t = *ui.DarkTheme()
	}
	if t.Dark {
		t.Background = ui.Hex("#192337")
		t.Surface = ui.Hex("#334463")
		t.SurfaceHover = ui.Hex("#465d7f")
		t.SurfacePressed = ui.Hex("#21304b")
		t.Text = ui.Hex("#fff1d5")
		t.TextMuted = ui.Hex("#b8c9d6")
		t.Border = ui.Hex("#719caa")
	} else {
		t.Background = ui.Hex("#efdbb8")
		t.Surface = ui.Hex("#fff0d4")
		t.SurfaceHover = ui.Hex("#ffe2a3")
		t.SurfacePressed = ui.Hex("#e6c292")
		t.Text = ui.Hex("#273349")
		t.TextMuted = ui.Hex("#4d6270")
		t.Border = ui.Hex("#53757b")
	}
	t.Accent = ui.Hex("#ffae75")
	t.AccentHover = ui.Hex("#ffca8f")
	t.AccentPressed = ui.Hex("#ed8c64")
	t.AccentText = ui.Hex("#27283f")
	t.Focus = ui.Hex("#75e2cf")
	t.Radius = 0
	t.FontSize = 12
	t.Spacing = 3
	if a.Typography == "Pixel" {
		t.Font = "Moonbounce Pixel"
	} else {
		t.Font = ""
	}
	c.SetTheme(&t)
	a.update(c)
	if c.Shortcut(ui.Cmd, ui.KeyS) {
		a.save()
	}
	if c.Shortcut(0, ui.KeyEscape) || c.Shortcut(0, ui.KeyP) {
		if d.maps || d.options {
			d.maps = false
			d.options = false
		} else {
			a.pause(!d.paused)
		}
		d.focus = true
	}

	stage := ui.Box(c.Key("road")).Fill().Clip().Focusable().AutoFocus().HandleInput(a.driveInput).Label(a.tr("Race: arrows drive, Shift drifts, Space boosts, Escape pauses", "赛车：方向键驾驶，Shift 漂移，空格加速，Esc 暂停", "賽車：方向鍵駕駛，Shift 漂移，空白鍵加速，Esc 暫停"))
	if d.focus {
		stage.Focus()
		d.focus = false
	}
	if !stage.Focused() {
		d.clearInput()
	}
	w, _ := c.Size()
	d.stageWidth = float64(w)
	stage.Children(func() {
		b := biomes[a.Journey.biome()]
		next := biomes[biomeFor(a.Journey.Seed, a.Journey.Region+1)]
		fade := float32(math.Max(0, (a.Journey.Offset-(regionLength-180))/180))
		ui.Image(c, d.skies[b.Sky]).Absolute().Top(0).LeftPercent(-10).WidthPercent(120).HeightPercent(36).Fit(ui.FillBox).PassThrough()
		if fade > 0 && next.Sky != b.Sky {
			ui.Image(c, d.skies[next.Sky]).Absolute().Top(0).LeftPercent(-10).WidthPercent(120).HeightPercent(36).Fit(ui.FillBox).Opacity(fade).PassThrough()
		}
		ui.Box(c).Absolute().Top(0).Bottom(0).Left(0).Right(0).PassThrough().Draw(func(p *ui.Painter, r ui.Rect) { a.paintDrive(p, r) })
		if d.maps {
			a.places(c)
		} else if d.options {
			a.options(c)
		} else if d.paused {
			a.pauseCard(c)
		} else {
			a.hud(c)
		}
		if a.status != "" || a.audioError != "" {
			ui.Text(c, a.status+" "+a.audioError).Absolute().Bottom(8).Left(12).Right(12).Padding(6).Background(t.Background).TextColor(t.Text).FontSize(11)
		}
	})
	if !d.paused && !d.maps && !d.options && !a.reduced {
		c.After(100 * time.Millisecond)
	}
}
func (d *DriveRuntime) place() int {
	place := 1
	for i := 0; i < 3; i++ {
		if d.traffic[i].Z > 3.8 {
			place++
		}
	}
	return place
}
func (a *Game) hud(c *ui.Context) {
	j, d := &a.Journey, &a.drive
	ink := ui.Hex("#fff1d5")
	bg := ui.RGBA(20, 28, 42, .83)
	ui.Row(c).Absolute().Top(16).Left(18).Right(18).Gap(12).Children(func() {
		ui.Text(c, fmt.Sprintf("%d / 4", d.place())).Font(a.valueFont()).FontSize(24).TextColor(ink).Background(bg).Padding(7, 10)
		ui.Text(c, fmt.Sprintf("%03.0f m", math.Max(0, sprintLength-d.race.Meters))).Font(a.valueFont()).FontSize(13).TextColor(ink).Background(bg).Padding(8)
		ui.Spacer(c)
		if gameButton(c, "pause", "II", false).Label(a.tr("Pause race", "暂停比赛", "暫停比賽")).Clicked() {
			a.pause(true)
			d.focus = true
		}
	})
	ui.Column(c).Absolute().Bottom(22).Right(22).Width(142).Gap(6).Padding(10).Background(bg).Children(func() {
		ui.Text(c, fmt.Sprintf("%.0f km/h", j.Speed*3.6)).Font(a.valueFont()).FontSize(23).TextColor(ink)
		ui.Box(c).Height(5).Background(ui.Hex("#566274")).Children(func() { ui.Box(c).Height(5).WidthPercent(float32(d.race.Nitro * 100)).Background(ui.Hex("#8ce8d0")) })
		label := a.tr("SPACE · BOOST", "空格 · 加速", "空白鍵 · 加速")
		if d.drafting {
			label = a.tr("DRAFTING +", "尾流蓄能 +", "尾流蓄能 +")
		} else if d.race.Nitro < .3 {
			label = a.tr("CHARGING", "蓄能中", "蓄能中")
		}
		ui.Text(c, label).FontSize(10).TextColor(ink)
	})
	if d.eventUntil > d.clock && d.event != 3 {
		msg := a.tr("OVERTAKE + BOOST", "超车 + 能量", "超車 + 能量")
		if d.event == 4 {
			msg = fmt.Sprintf(a.tr("FINISH %d / 4 · NEXT RACE →", "完赛 %d / 4 · 下一程 →", "完賽 %d / 4 · 下一程 →"), d.race.LastPlace)
		}
		ui.Text(c, msg).Absolute().Top(72).Left(18).Padding(8).Background(bg).TextColor(ink).FontSize(14)
	}
}
func (a *Game) pauseCard(c *ui.Context) {
	d := &a.drive
	ui.Column(c).Absolute().TopPercent(22).LeftPercent(30).RightPercent(30).Padding(24).Gap(16).Background(ui.RGBA(20, 28, 42, .96)).Children(func() {
		ui.Text(c, "MOONBOUNCE").FontSize(26).TextColor(ui.Hex("#fff1d5"))
		b := biomes[a.Journey.biome()]
		ui.Text(c, fmt.Sprintf("%s · %.1f km", a.tr(b.Name, b.Hans, b.Hant), a.Journey.Kilometers)).FontSize(12).TextColor(ui.Hex("#afcfcc"))
		if gameButton(c, "start", a.tr("RACE →", "开赛 →", "開賽 →"), true).Height(46).Disabled(a.reduced).Clicked() {
			a.pause(false)
			d.focus = true
		}
		ui.Row(c).Gap(8).Children(func() {
			if gameButton(c, "places", a.tr("ROUTE", "路线", "路線"), false).Grow(1).Clicked() {
				d.maps = true
			}
			if gameButton(c, "options", a.tr("SETTINGS", "设置", "設定"), false).Grow(1).Clicked() {
				d.options = true
			}
		})
		ui.Text(c, a.tr("↑ gas  ↓ brake  ← → steer", "↑ 油门  ↓ 刹车  ← → 转向", "↑ 油門  ↓ 煞車  ← → 轉向")).FontSize(11).TextColor(ui.Hex("#afcfcc"))
		ui.Text(c, a.tr("Shift drift · Space boost · Esc pause", "Shift 漂移 · 空格加速 · Esc 暂停", "Shift 漂移 · 空白鍵加速 · Esc 暫停")).FontSize(10).TextColor(ui.Hex("#afcfcc"))
		if a.reduced {
			ui.Text(c, a.tr("Reduce Motion is on. Choose a route to explore.", "已减弱动态效果，可选择路线。", "已減少動態效果，可選擇路線。")).FontSize(11).TextColor(ui.Hex("#fff1d5"))
		}
	})
}
func (a *Game) places(c *ui.Context) {
	ui.Column(c).Absolute().Top(14).Bottom(14).Left(14).Right(14).Padding(20).Gap(12).Background(ui.RGBA(25, 35, 55, .98)).Border(3, ui.Hex("#91d4c7")).Children(func() {
		ui.Row(c).Gap(12).Children(func() {
			ui.Text(c, a.tr("PICK A POSTCARD", "下一张明信片", "下一張明信片")).FontSize(23).TextColor(ui.Hex("#fff0ca"))
			ui.Spacer(c)
			if gameButton(c, "places-close", a.tr("BACK", "返回", "返回"), false).Clicked() {
				a.drive.maps = false
				a.drive.focus = true
			}
		})
		ui.Text(c, a.tr("Every place is open. The road keeps making more.", "每个地方都已开放，沿途总有新风景。", "每個地方都已開放，沿途總有新風景。")).FontSize(12).TextColor(ui.Hex("#b9d9d6"))
		for row := 0; row < 4; row++ {
			ui.Row(c).Grow(1).Gap(9).Children(func() {
				for col := 0; col < 4; col++ {
					idx := row*4 + col
					b := biomes[idx]
					label := a.tr(b.Name, b.Hans, b.Hant)
					if a.Journey.Visited&(1<<idx) != 0 {
						label = "• " + label
					}
					if gameButton(c, fmt.Sprintf("world-%d", idx), label, false).Grow(1).MinWidth(0).Height(62).Background(rgb(b.Road)).BorderColor(rgb(b.Accent)).TextColor(ui.Hex("#fff2d7")).FontSize(12).Clicked() {
						a.Journey.visit(idx)
						a.drive.maps = false
						a.drive.focus = true
						a.drive.initTraffic(a.Journey.Seed + a.Journey.Region)
						a.drive.race.Meters = 0
						a.drive.event = 3
						a.drive.eventUntil = a.drive.clock + 3
						a.save()
					}
				}
			})
		}
		ui.Row(c).Children(func() {
			ui.Text(c, fmt.Sprintf(a.tr("%d / 16 POSTCARDS", "%d / 16 张明信片", "%d / 16 張明信片"), bits.OnesCount32(a.Journey.Visited))).FontSize(11).TextColor(ui.Hex("#b9d9d6"))
			ui.Spacer(c)
			if gameButton(c, "surprise", a.tr("SURPRISE ME", "随心漫游", "隨心漫遊"), true).Clicked() {
				a.Journey.visit((a.Journey.biome() + 1 + int(mix64(a.Journey.Region)%15)) % 16)
				a.drive.maps = false
				a.drive.focus = true
				a.drive.initTraffic(a.Journey.Seed + a.Journey.Region)
				a.drive.race.Meters = 0
				a.save()
			}
		})
	})
}
func (a *Game) options(c *ui.Context) {
	ui.Column(c).Absolute().TopPercent(10).Height(500).LeftPercent(20).RightPercent(20).Padding(20).Gap(10).Background(c.Theme().Background).Border(3, c.Theme().Accent).Children(func() {
		ui.Row(c).Children(func() {
			ui.Text(c, a.tr("SETTINGS", "设置", "設定")).FontSize(20)
			ui.Spacer(c)
			if gameButton(c, "options-close", a.tr("BACK", "返回", "返回"), false).Clicked() {
				a.drive.options = false
				a.drive.focus = true
			}
		})
		ui.Row(c).Gap(6).Children(func() {
			for i, f := range feels {
				if gameButton(c, fmt.Sprintf("feel-%d", i), a.tr(f.Name, f.Hans, f.Hant), a.Journey.Feel == i).Grow(1).Clicked() {
					a.Journey.Feel = i
					a.save()
				}
			}
		})

		ui.Row(c).Gap(10).Children(func() {
			ui.Text(c, a.tr("Language", "语言", "語言")).Width(115)
			if gameButton(c, "language", a.Language, false).Grow(1).Clicked() {
				switch a.Language {
				case "English":
					a.Language = "简体中文"
				case "简体中文":
					a.Language = "繁體中文"
				default:
					a.Language = "English"
				}
				a.save()
			}
		})
		ui.Row(c).Gap(10).Children(func() {
			ui.Text(c, a.tr("Typography", "字体", "字體")).Width(115)
			if gameButton(c, "font", a.fontLabel(), a.Typography == "Pixel").Grow(1).Clicked() {
				if a.Typography == "Pixel" {
					a.Typography = "System"
				} else {
					a.Typography = "Pixel"
				}
				a.save()
			}
		})
		theme := a.tr("System", "跟随系统", "跟隨系統")
		if a.Appearance == "Light" {
			theme = a.tr("Daylight", "日光", "日光")
		}
		if a.Appearance == "Dark" {
			theme = a.tr("Midnight", "午夜", "午夜")
		}
		ui.Row(c).Gap(10).Children(func() {
			ui.Text(c, a.tr("Cabin", "车内配色", "車內配色")).Width(115)
			if gameButton(c, "theme", theme, false).Grow(1).Clicked() {
				switch a.Appearance {
				case "System":
					a.Appearance = "Dark"
				case "Dark":
					a.Appearance = "Light"
				default:
					a.Appearance = "System"
				}
				a.save()
			}
		})
		cruise := a.tr("Cruise throttle: ON · C", "巡航油门：开 · C", "巡航油門：開 · C")
		if !a.Cruise {
			cruise = a.tr("Manual throttle · C", "手动油门 · C", "手動油門 · C")
		}
		if gameButton(c, "cruise", cruise, a.Cruise).Clicked() {
			a.Cruise = !a.Cruise
			a.save()
		}
		if a.radio != nil {
			effects := a.tr("Engine & tires: ON", "引擎与轮胎：开", "引擎與輪胎：開")
			if !a.Audio.Effects {
				effects = a.tr("Engine & tires: OFF", "引擎与轮胎：关", "引擎與輪胎：關")
			}
			if gameButton(c, "effects", effects, a.Audio.Effects).Clicked() {
				a.radio.toggleEffects()
				a.save()
			}
		}
		ui.Spacer(c)
		ui.Text(c, a.tr("RADIO: Holizna · Lo-Fi and Chill · CC0", "电台：Holizna · Lo-Fi and Chill · CC0", "電台：Holizna · Lo-Fi and Chill · CC0")).FontSize(11).TextColor(c.Theme().TextMuted)
		a.radioPanel(c)
	})
}
func (a *Game) radioPanel(c *ui.Context) {
	ui.Column(c).Height(84).Padding(10).Gap(8).Background(c.Theme().Surface).Border(2, c.Theme().Border).Children(func() {
		title := a.tr("RADIO / ", "电台 / ", "電台 / ") + tracks[a.Audio.Track].Title
		if a.audioError != "" {
			title = a.tr("Audio unavailable: ", "音频不可用：", "音訊無法使用：") + a.audioError
		}
		ui.Row(c).Gap(4).Children(func() {
			ui.Text(c, title).FontSize(11).TextColor(c.Theme().Text)
			ui.Spacer(c)
			seconds := int(math.Max(0, a.drive.musicSeconds))
			state := fmt.Sprintf("%d:%02d  %.0f%%", seconds/60, seconds%60, a.Audio.Volume*100)
			if !a.Audio.Music {
				state = a.tr("PAUSED", "暂停", "暫停")
				if a.Audio.Stopped {
					state = a.tr("STOPPED", "停止", "停止")
				}
			} else if a.drive.paused || a.drive.maps || a.drive.options {
				state = a.tr("PARKED", "已停靠", "已停靠")
			}
			ui.Text(c, state).Font(a.valueFont()).FontSize(10).TextColor(c.Theme().TextMuted)
		})
		ui.Row(c).Gap(5).Children(func() {
			play := ">"
			if a.Audio.Music {
				play = "II"
			}
			if gameButton(c, "radio-play", play, a.Audio.Music).Label(a.tr("Play or pause radio", "播放或暂停电台", "播放或暫停電台")).Disabled(a.radio == nil).Clicked() {
				a.radio.toggle()
				a.save()
				a.drive.focus = true
			}
			if gameButton(c, "radio-stop", a.tr("STOP", "停止", "停止"), false).Disabled(a.radio == nil).Clicked() {
				a.audioAction(a.radio.stop)
			}
			if gameButton(c, "radio-replay", "|<", false).Label(a.tr("Replay track from start", "从头重播", "從頭重播")).Disabled(a.radio == nil).Clicked() {
				a.audioAction(a.radio.replay)
			}
			if gameButton(c, "radio-next", ">|", false).Label(a.tr("Next track", "下一首", "下一首")).Disabled(a.radio == nil).Clicked() {
				a.audioAction(a.radio.next)
			}
			ui.Spacer(c)
			if gameButton(c, "volume-down", "−", false).Padding(6).Label(a.tr("Music volume down", "降低音乐音量", "降低音樂音量")).Disabled(a.radio == nil).Clicked() {
				a.radio.setVolume(a.Audio.Volume - .1)
				a.save()
			}
			if gameButton(c, "volume-up", "+", false).Padding(6).Label(a.tr("Music volume up", "提高音乐音量", "提高音樂音量")).Disabled(a.radio == nil).Clicked() {
				a.radio.setVolume(a.Audio.Volume + .1)
				a.save()
			}
		})
	})
}

func (a *Game) prepareRoad() {
	d, j := &a.drive, &a.Journey
	heading, x := 0.0, 0.0
	d.roadX[0] = 0
	for i := 1; i < len(d.roadX); i++ {
		curvature := j.curve(float64(i)*10) * .0015
		heading += curvature * 10
		x += heading * 10
		d.roadX[i] = x
	}
}
func (a *Game) project(z, w, h float64) (x, y, half float64) {
	z = math.Max(3, z)
	at := math.Min(z/10, float64(len(a.drive.roadX)-2))
	i := int(at)
	f := at - float64(i)
	world := a.drive.roadX[i]*(1-f) + a.drive.roadX[i+1]*f
	yaw := a.Journey.curve(0)*.0015*32 + a.Journey.Steering*.018
	scale := .85 / z
	x = w*.5 + (world-yaw*z-a.Journey.X*7.5)*scale*w*.5
	y = h*.36 + 4.8*scale*h*.5
	half = 7.5 * scale * w * .5
	return
}
func (a *Game) paintDrive(p *ui.Painter, r ui.Rect) {
	a.advance(p.Now())
	if !a.drive.paused && !a.drive.maps && !a.drive.options && !a.reduced {
		p.AnimationFrame()
	}
	d, j := &a.drive, &a.Journey
	if !p.Now().Equal(d.lastPaint) {
		if !d.lastPaint.IsZero() && !d.paused && !d.maps && !d.options {
			ms := p.Now().Sub(d.lastPaint).Seconds() * 1000
			d.paintIntervals = append(d.paintIntervals, ms)
			if len(d.paintIntervals) > 7200 {
				d.paintIntervals = d.paintIntervals[1:]
			}
		}
		d.lastPaint = p.Now()
		d.painted++
	}
	a.prepareRoad()
	b := biomes[j.biome()]
	next := biomes[biomeFor(j.Seed, j.Region+1)]
	blend := float32(math.Max(0, (j.Offset-(regionLength-180))/180))
	ground := rgb(b.Ground).Mix(rgb(next.Ground), blend)
	alt := rgb(b.Alternate).Mix(rgb(next.Alternate), blend)
	road := rgb(b.Road).Mix(rgb(next.Road), blend)
	edge := rgb(b.Edge).Mix(rgb(next.Edge), blend)
	w, h := float64(r.W), float64(r.H)
	horizon := h * .36
	pixel := math.Max(2, math.Ceil(w/420))
	p.Clip(r, 0, func() {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + float32(horizon), W: r.W, H: r.H - float32(horizon)}, ground, 0)

		for yy := horizon; yy < h; yy += pixel {
			z := 4.8 * .85 * h * .5 / math.Max(yy-horizon, 1)
			z = math.Min(z, 1550)
			center, _, half := a.project(z, w, h)
			rh := float32(pixel + 1)
			y := r.Y + float32(yy)
			stripe := int((j.Offset+z)/7)%2 == 0
			if stripe {
				p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: rh}, alt, 0)
			}
			shoulder := ui.Hex("#e96667")
			if !stripe {
				shoulder = ui.Hex("#f2eeee")
			}
			p.Fill(ui.Rect{X: r.X + float32(math.Round((center-half*1.07)/pixel)*pixel), Y: y, W: float32(half * 2.14), H: rh}, shoulder, 0)
			p.Fill(ui.Rect{X: r.X + float32(math.Round((center-half)/pixel)*pixel), Y: y, W: float32(half * 2), H: rh}, road, 0)
			if stripe && z < 1100 {
				for _, lane := range []float64{-.333, .333} {
					p.Fill(ui.Rect{X: r.X + float32(center+half*lane), Y: y, W: float32(math.Max(pixel, half*.014)), H: rh}, edge, 0)
				}
			}
			if math.Abs(z-(sprintLength-d.race.Meters+3.8)) < 2 {
				for cell := 0; cell < 12; cell++ {
					color := ui.Hex("#fff4d9")
					if cell%2 == int((z-(sprintLength-d.race.Meters)+2)/2)%2 {
						color = ui.Hex("#202c40")
					}
					p.Fill(ui.Rect{X: r.X + float32(center-half+float64(cell)*half/6), Y: y, W: float32(half/6 + 1), H: rh}, color, 0)
				}
			}
			if j.roadStyle() == 3 {
				for _, side := range []float64{-1, 1} {
					p.Fill(ui.Rect{X: r.X + float32(center+side*half*1.15), Y: y, W: float32(math.Max(pixel, half*.035)), H: rh}, rgb(b.Accent), 0)
				}
			}
		}
		// Back-to-front world props use the same perspective as the road.
		for n := 48; n >= 0; n-- {
			z := float64(n)*24 - math.Mod(j.Offset, 24) + 10
			if z < 8 {
				continue
			}
			x, y, half := a.project(z, w, h)
			size := math.Min(250, half*.33)
			id := j.Region*100 + uint64(math.Floor(j.Offset/24)) + uint64(n)
			for _, side := range []float64{-1, 1} {
				dist := 1.28 + random01(id)*.55
				a.drawProp(p, float64(r.X)+x+side*half*dist, float64(r.Y)+y, size, b, id)
			}
		}
		// Draw distant cars first, without sorting or allocating a variable queue.
		order := [6]int{0, 1, 2, 3, 4, 5}
		for i := 0; i < 6; i++ {
			for k := i + 1; k < 6; k++ {
				if d.traffic[order[k]].Z > d.traffic[order[i]].Z {
					order[i], order[k] = order[k], order[i]
				}
			}
		}
		for _, idx := range order {
			car := d.traffic[idx]
			if car.Z < 1.8 || car.Z > 1200 {
				continue
			}
			x, y, half := a.project(car.Z, w, h)
			size := half * .14
			img := d.trafficCar
			if idx < 3 {
				img = d.cars[0]
			}
			if img == nil {
				img = d.cars[0]
			}
			p.Image(img, ui.Rect{X: r.X + float32(x+car.X*half-size*.65), Y: r.Y + float32(y-size*.85), W: float32(size * 1.3), H: float32(size * .9)}, ui.Contain)
		}
		carW := math.Min(180, w*.155)
		carH := carW * .67
		carX := w*.5 + j.Steering*w*.012
		carY := h*.91 + math.Sin(d.clock*17)*math.Min(j.Speed/55, 1)*1.5
		if d.drift && math.Abs(j.Steering) > .15 && j.Speed > 15 {
			for n := 0; n < 9; n++ {
				life := math.Mod(d.clock*5+float64(n)*.13, 1)
				side := 1.0
				if n%2 == 0 {
					side = -1
				}
				size := 4 + life*12
				p.Fill(ui.Rect{X: r.X + float32(carX+side*carW*.31+j.Lateral*life*28), Y: r.Y + float32(carY+life*26-10), W: float32(size), H: float32(size)}, ui.RGBA(234, 218, 202, float32((1-life)*.65)), 0)
			}
		}
		frame := 0
		if j.Steering < -.16 {
			frame = 1
		} else if j.Steering > .16 {
			frame = 2
		}
		if d.drift && frame > 0 {
			frame += 3
		}
		if d.brake && frame == 0 {
			frame = 3
		}
		p.Fill(ui.Rect{X: r.X + float32(carX-carW*.41), Y: r.Y + float32(carY-5), W: float32(carW * .82), H: 8}, ui.RGBA(17, 22, 38, .35), 0)
		p.Image(d.cars[frame], ui.Rect{X: r.X + float32(carX-carW/2), Y: r.Y + float32(carY-carH), W: float32(carW), H: float32(carH)}, ui.Contain)
		if j.Boost > 0 {
			for n := 0; n < 12; n++ {
				u := math.Mod(d.clock*2+float64(n)*.083, 1)
				side := float64(n%2)*2 - 1
				x := w*.5 + side*(w*.33+u*w*.20)
				p.Fill(ui.Rect{X: r.X + float32(x), Y: r.Y + float32(h*.45+u*h*.55), W: float32(pixel), H: float32(8 + u*34)}, ui.RGBA(255, 238, 186, float32(u*.45)), 0)
			}
		}
	})
}
func (a *Game) drawProp(p *ui.Painter, x, y, s float64, b Biome, id uint64) {
	if s < 2 {
		return
	}
	height := s * 1.9
	w := height
	if b.Prop == 7 || b.Prop == 8 {
		height *= .60
	}
	if b.Prop == 5 {
		height *= 1.15
	}
	p.Image(a.drive.props[b.Prop], ui.Rect{X: float32(x - w/2), Y: float32(y - height), W: float32(w), H: float32(height)}, ui.Contain)
}

func (a *Game) valueFont() string {
	if a.Typography == "Pixel" {
		return "Moonbounce Pixel"
	}
	return "monospace"
}
