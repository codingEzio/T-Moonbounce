# Moonbounce

[English](README.md) · **简体中文** · [繁體中文](README.zh-TW.md)

一款小巧的原生 macOS 街机赛车游戏。沿着无尽的像素公路追逐三位对手，参加每公里一场的竞速，听离线 lo-fi 电台。

![Moonbounce：海岸像素赛车](docs/moonbounce.webp)

**竞速、漂移，再来一场。** 跟车、超车和漂移可积累加速能量。16 条路线和四种驾驶手感从一开始就全部开放。驶入草地或发生碰撞会损失速度，但不会清空累计行程。行驶画面仅保留名次、距离、速度/能量和暂停按钮。

## 运行

需要 **macOS 14+** 和 **Go 1.27.1+**。已在 Apple Silicon 上验证；本项目不支持其他平台。

```sh
git clone https://github.com/codingEzio/T-Moonbounce.git
cd T-Moonbounce
./run
```

构建独立应用：

```sh
./package
open .local/Moonbounce.app
```

首次构建会下载固定版本的 Go 模块。应用内置美术、字体和三首音乐，可离线使用。本机构建的应用采用临时签名（ad-hoc），未经 Apple 公证。

## 操作

| 按键 | 操作 |
| --- | --- |
| ← → / A D | 转向 |
| ↑ / W | 加速 |
| ↓ / S | 刹车 |
| Shift + 转向 | 漂移 |
| 空格 | 消耗能量加速 |
| Escape / P | 暂停或继续；优先关闭子菜单 |
| C | 切换巡航油门 |

默认使用 **Rally** 和手动油门，也可在设置中选择 Sunday、Cloud 或 Comet。巡航负责油门，你负责转向。按住或拖动路面可用指针转向。

暂停后可进入 **Race · Route · Settings**。设置提供英文、简体中文和繁体中文，像素/系统字体，浅色/深色/跟随系统外观，以及驾驶和音频选项。系统“减少动态效果”开启时，仍可浏览路线，但不会播放动画。

电台收录 Holizna 的 **Windows Down**、**Keeping Cool** 和 **First Snow**。游戏暂停或打开菜单时，音乐也会暂停。电台暂停保留播放位置，停止会归零，重播从头开始，下一首切换曲目。详见[音乐说明](assets/audio/CREDITS.md)。

## 本机数据与隐私

无需账号，没有分析、遥测或运行时网络请求。普通应用构建把行程和偏好保存在 `~/Library/Application Support/Moonbounce/settings.json`。`./run` 使用项目内已被 Git 忽略的 `Local/settings.json`；`--state <path>` 可指定文件。这些是独立存档。转移开发存档时，先关闭两个实例，再复制到应用的数据文件夹，并保留原文件。

存档采用原子写入并在写入后核验。之前的有效存档和无法读取的原文件会保留以便恢复。比赛名次和加速能量仅在当前会话中有效；距离、探索、超车、漂移和偏好会持久保存。开发缓存、存档和私人记录均不纳入 Git。

## 开发

使用 [MyGo](https://github.com/egoist/mygo) 和 Metal 渲染，[Oto](https://github.com/ebitengine/oto) 提供原生音频，[go-mp3](https://github.com/hajimehoshi/go-mp3) 解码音乐。渲染采用实际经过时间和原生显示回调，没有固定 60 Hz 的模拟步长。

```sh
go test ./...
go vet ./...
# 打开原生窗口，使用真实音频传输但输出音量为零：
./run --mute-output --state .local/check.json --benchmark 18s --capture .local/check.png
# 可选的 CoreAudio 传输检查，同样静音：
MOONBOUNCE_AUDIO_TEST=1 MOONBOUNCE_MUTE_OUTPUT=1 go test -run TestNativeAudioTransport -v
```

原生检查需要图形化 macOS 会话。普通测试不会打开窗口或播放音频。GitHub Actions 检查格式、测试、vet 和应用打包；它不证明视觉质量、实际屏幕刷新率或听感。详见[贡献说明](CONTRIBUTING.md)。

## 许可与致谢

项目代码和原创生成的游戏美术，在适用权利存在的范围内采用 [MIT](LICENSE) 许可。截图来自实际应用窗口，保存为无元数据的无损 WebP。

内置音乐使用 **CC0**。修改后的字体及其组件保留各自许可，不重新授权为 MIT。仓库包含依赖声明、来源信息和图像生成提示词。详见[依赖与资产](DEPENDENCIES.md)、[字体说明](assets/fonts/NOTICE.md)和[美术提示词](assets/PROMPTS.md)。
