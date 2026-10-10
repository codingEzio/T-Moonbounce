# Dependencies and assets

Go modules are pinned in `go.mod` and `go.sum`. They are downloaded during the first build and linked into the executable; gameplay does not require the network. Complete available upstream notices are bundled under `assets/licenses/`.

| Component | Version | License | Purpose / source |
| --- | --- | --- | --- |
| MyGo | 0.3.7 | MIT | Native window, input, text and Metal rendering · https://github.com/egoist/mygo |
| Oto | 3.5.0 | Apache-2.0 | Native CoreAudio playback · https://github.com/ebitengine/oto |
| go-mp3 | 0.3.4 | Apache-2.0 | Streaming MP3 decoding and seek · https://github.com/hajimehoshi/go-mp3 |
| purego | 0.11.1 | Apache-2.0 | Native calls without cgo · https://github.com/ebitengine/purego |
| go-text/typesetting | 0.3.5 | MIT / Unlicense | Text shaping · https://github.com/go-text/typesetting |
| golang.org/x/image | 0.46.0 | BSD-3-Clause | Image support · https://cs.opensource.google/go/x/image |
| golang.org/x/sys | 0.48.0 | BSD-3-Clause | Platform support · https://cs.opensource.google/go/x/sys |
| jfreymuth/pulse | 0.1.3 | MIT | Oto's Linux module dependency; unused by this macOS product · https://github.com/jfreymuth/pulse |

## Bundled assets

- **Game artwork:** `car.png`, `hatchback-sheet.png`, `horizon-sheet.png`, `roadside-sheet.png`. Original OpenAI-generated art for Moonbounce; project contributions offered under MIT to the extent applicable rights exist. Exact [prompts](assets/PROMPTS.md) are included. Original PNGs are embedded; atlas cropping and coarse pixel rasterization happen at load time.
- **Music:** three unmodified MP3s by Holizna from *Lo-Fi and Chill*, explicitly released under CC0 1.0. See [source, original filenames and license](assets/audio/CREDITS.md). Tracks are embedded and decoded locally.
- **DeparturePixelZh Compact:** modified font from [X-DeparturePixelZh](https://github.com/codingEzio/X-DeparturePixelZh), SHA-256 `3e75e623b1ba2c8d7eb52f2ac3ee71c3fbb5c4e0f1426bd34b3c1e071738288b`. Bundles Departure Mono, Cubic 11 and Nerd Fonts components. The derived font uses OFL 1.1, with retained component notices and applicable original grants. Full [notices](assets/fonts/NOTICE.md), licenses and public provenance are included. Registered as Moonbounce Pixel; System sans/mono is an optional preference. No OS fonts are redistributed.
- **README screenshot:** actual native app capture from isolated synthetic state, losslessly compressed to WebP without metadata. Capture and compression details are in [screenshot receipt](docs/screenshot.json).

`assets/manifest.json` records the artwork and music digests. Assets are included in the executable, with no runtime download or paid service. The root MIT license does not replace any third-party license.
