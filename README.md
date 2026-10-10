# Moonbounce

**English** · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md)

A tiny native macOS arcade racer. Chase three rivals through an endless pixel road trip, with one-kilometer sprints and an offline lo-fi radio.

![Moonbounce: pixel racing on the coast](docs/moonbounce.webp)

**Race, drift, repeat.** Draft behind rivals, overtake and drift to earn boost. Sixteen routes and four handling styles are open from the start. Grass and contact cost momentum; your accumulated journey stays yours. While racing, the screen shows only position, distance, speed/boost and Pause.

## Run

Requires **macOS 14+** and **Go 1.27.1+**. Apple Silicon is verified; other platforms are not supported by this project.

```sh
git clone https://github.com/codingEzio/T-Moonbounce.git
cd T-Moonbounce
./run
```

To build a self-contained app:

```sh
./package
open .local/Moonbounce.app
```

The first build downloads pinned Go modules. The app includes its artwork, font and three music tracks and works offline. The locally built app is ad-hoc signed, not notarized.

## Controls

| Key | Action |
| --- | --- |
| ← → / A D | Steer |
| ↑ / W | Accelerate |
| ↓ / S | Brake |
| Shift + steering | Drift |
| Space | Spend energy on boost |
| Escape / P | Pause or resume; close a submenu first |
| C | Toggle cruise throttle |

Start in **Rally** with manual throttle, or choose Sunday, Cloud or Comet in Settings. Cruise handles the throttle; you steer. Hold or drag on the road for pointer steering.

Pause opens **Race · Route · Settings**. Settings includes English, 简体中文 and 繁體中文; Pixel/System typography; light/dark/system appearance; handling; and audio controls. System Reduce Motion keeps route browsing available without animation.

The radio plays **Windows Down**, **Keeping Cool** and **First Snow** by Holizna. Game pause and menus pause the music. Radio pause retains its position, Stop resets it, Replay starts from zero, and Next selects another track. See [music credits](assets/audio/CREDITS.md).

## Local data and privacy

No accounts, analytics, telemetry or runtime network requests. Normal app builds store the journey and preferences in `~/Library/Application Support/Moonbounce/settings.json`. `./run` uses the checkout's ignored `Local/settings.json`; `--state <path>` selects an explicit file. These are separate saves. To transfer a development save, close both instances and copy it to the app's data folder, keeping the original.

Saves are atomic and checked after writing. Previous valid saves and unreadable originals are retained for recovery. Race position and boost energy are session-local; distance, discoveries, passes, drift and preferences persist. Development caches, saves and private notes are excluded from Git.

## Development

Built with [MyGo](https://github.com/egoist/mygo) and Metal, [Oto](https://github.com/ebitengine/oto) for native audio, and [go-mp3](https://github.com/hajimehoshi/go-mp3). Rendering uses elapsed time and native display callbacks; there is no fixed 60 Hz simulation step.

```sh
go test ./...
go vet ./...
# Visible native check with real audio transport but zero output gain:
./run --mute-output --state .local/check.json --benchmark 18s --capture .local/check.png
# Opt-in CoreAudio transport check, also muted:
MOONBOUNCE_AUDIO_TEST=1 MOONBOUNCE_MUTE_OUTPUT=1 go test -run TestNativeAudioTransport -v
```

Native checks require a graphical macOS session. Ordinary tests do not open windows or play audio. GitHub Actions checks formatting, tests, vet and app packaging; it does not establish visual quality, physical refresh rate or listening quality. [Contributing](CONTRIBUTING.md).

## License and credits

Project code and original generated game art are offered under [MIT](LICENSE), to the extent applicable rights exist. The screenshot is an actual app capture, stored as lossless, metadata-free WebP.

Bundled music is **CC0**. The modified font and its component notices retain their own licenses; they are not relicensed under MIT. Dependency notices, provenance and image-generation prompts are included. See [dependencies and assets](DEPENDENCIES.md), [font notices](assets/fonts/NOTICE.md) and [art prompts](assets/PROMPTS.md).
