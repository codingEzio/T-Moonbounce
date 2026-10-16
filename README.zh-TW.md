# Moonbounce

[English](README.md) · [简体中文](README.zh-CN.md) · **繁體中文**

一款小巧的原生 macOS 街機賽車遊戲。沿著無盡的像素公路追逐三位對手，參加每公里一場的競速，聽離線 lo-fi 電台。

![Moonbounce：海岸像素賽車](docs/moonbounce.webp)

**競速、甩尾，再來一場。** 跟車、超車與甩尾可累積加速能量。16 條路線與四種駕駛手感從一開始就全部開放。駛入草地或發生碰撞會損失速度，但不會清空累計行程。行駛畫面僅保留名次、距離、速度/能量與暫停按鈕。

## 執行

需要 **macOS 14+** 與 **Go 1.27.1+**。已在 Apple Silicon 上驗證；本專案不支援其他平台。

```sh
git clone https://github.com/codingEzio/T-Moonbounce.git M-Moonbounce
cd M-Moonbounce
./run
```

建置獨立應用程式：

```sh
./package
open .local/Moonbounce.app
```

首次建置會下載固定版本的 Go 模組。應用程式內建美術、字型與三首音樂，可離線使用。本機建置的應用程式採用臨時簽署（ad-hoc），未經 Apple 公證。

## 操作

| 按鍵 | 操作 |
| --- | --- |
| ← → / A D | 轉向 |
| ↑ / W | 加速 |
| ↓ / S | 煞車 |
| Shift + 轉向 | 甩尾 |
| 空白鍵 | 消耗能量加速 |
| Escape / P | 暫停或繼續；優先關閉子選單 |
| C | 切換巡航油門 |

預設使用 **Rally** 與手動油門，也可在設定中選擇 Sunday、Cloud 或 Comet。巡航負責油門，你負責轉向。按住或拖曳路面可用指標轉向。

暫停後可進入 **Race · Route · Settings**。設定提供英文、簡體中文與繁體中文，像素/系統字型，淺色/深色/跟隨系統外觀，以及駕駛與音訊選項。系統「減少動態效果」開啟時，仍可瀏覽路線，但不會播放動畫。

電台收錄 Holizna 的 **Windows Down**、**Keeping Cool** 與 **First Snow**。遊戲暫停或開啟選單時，音樂也會暫停。電台暫停保留播放位置，停止會歸零，重播從頭開始，下一首切換曲目。詳見[音樂說明](assets/audio/CREDITS.md)。

## 本機資料與隱私

無需帳號，沒有分析、遙測或執行時網路請求。一般應用程式建置把行程與偏好儲存在 `~/Library/Application Support/Moonbounce/settings.json`。`./run` 使用專案內已被 Git 忽略的 `Local/settings.json`；`--state <path>` 可指定檔案。這些是獨立存檔。轉移開發存檔時，先關閉兩個執行個體，再複製到應用程式的資料資料夾，並保留原檔案。

存檔採用原子寫入並在寫入後核驗。之前的有效存檔與無法讀取的原檔案會保留以便復原。比賽名次與加速能量僅在目前工作階段中有效；距離、探索、超車、甩尾與偏好會持久儲存。開發快取、存檔與私人記錄均不納入 Git。

## 開發

使用 [MyGo](https://github.com/egoist/mygo) 與 Metal 繪製，[Oto](https://github.com/ebitengine/oto) 提供原生音訊，[go-mp3](https://github.com/hajimehoshi/go-mp3) 解碼音樂。繪製採用實際經過時間與原生顯示回呼，沒有固定 60 Hz 的模擬步長。

```sh
go test ./...
go vet ./...
# 開啟原生視窗，使用真實音訊傳輸但輸出音量為零：
./run --mute-output --state .local/check.json --benchmark 18s --capture .local/check.png
# 可選的 CoreAudio 傳輸檢查，同樣靜音：
MOONBOUNCE_AUDIO_TEST=1 MOONBOUNCE_MUTE_OUTPUT=1 go test -run TestNativeAudioTransport -v
```

原生檢查需要圖形化 macOS 工作階段。一般測試不會開啟視窗或播放音訊。GitHub Actions 檢查格式、測試、vet 與應用程式打包；它不證明視覺品質、實際螢幕更新率或聽感。詳見[貢獻說明](CONTRIBUTING.md)。

## 授權與致謝

專案程式碼與原創生成的遊戲美術，在適用權利存在的範圍內採用 [MIT](LICENSE) 授權。截圖來自實際應用程式視窗，儲存為無中繼資料的無損 WebP。

內建音樂使用 **CC0**。修改後的字型及其元件保留各自授權，不重新授權為 MIT。儲存庫包含相依套件聲明、來源資訊與影像生成提示詞。詳見[相依套件與資產](DEPENDENCIES.md)、[字型說明](assets/fonts/NOTICE.md)與[美術提示詞](assets/PROMPTS.md)。
