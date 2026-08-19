<p align="center">
  <img src="doc/logo.png" alt="spank logo" width="200">
</p>

# spank

[English][readme-en-link] | **简体中文**

拍打你的 MacBook，它会回应你。

> "这是我见过的最神奇的东西" — [@kenwheeler](https://x.com/kenwheeler)

> "我刚才在妻子旁边运行了性感模式...笑死我们了" — [@duncanthedev](https://x.com/duncanthedev)

> "巅峰工程" — [@tylertaewook](https://x.com/tylertaewook)

使用 Apple Silicon 芯片的加速度传感器，检测笔记本电脑的物理撞击并播放音频回应。单个二进制文件，无依赖。

## 系统要求

- 基于 Apple Silicon 的 macOS（任何 M2 及以上型号的 M 系列芯片，或特定的 M1 Pro 型号，不包括其他 M1/A 系列芯片！）
- `sudo`（用于访问加速度传感器）
- Go 1.26+（如果从源代码构建）

## 安装

从[最新版本](https://github.com/taigrr/spank/releases/latest)下载。

或者从源代码构建：

```bash
go install github.com/taigrr/spank@latest
```

> **注意：** `go install` 会将二进制文件放在 `$GOBIN`（如果已设置）或 `$(go env GOPATH)/bin`（默认为 `~/go/bin`）。将其复制到系统路径以便 `sudo spank` 能够工作。例如，使用默认的 Go 设置：
>
> ```bash
> sudo cp "$(go env GOPATH)/bin/spank" /usr/local/bin/spank
> ```

## 使用方法

```bash
# 普通模式 — 拍打时说"ow!"
sudo spank

# 性感模式 — 根据拍打频率升级回应
sudo spank --sexy

# 光环模式 — 拍打时播放光环死亡音效
sudo spank --halo

# 快速模式 — 更快的轮询和更短的冷却时间
sudo spank --fast
sudo spank --sexy --fast

# 自定义模式 — 从指定目录播放你自己的 MP3 文件
sudo spank --custom /path/to/mp3s

# 使用振幅阈值调整灵敏度（数值越低越敏感）
sudo spank --min-amplitude 0.1   # 更敏感
sudo spank --min-amplitude 0.25  # 不太敏感
sudo spank --sexy --min-amplitude 0.2

# 设置冷却时间（毫秒，默认：750）
sudo spank --cooldown 600

# 设置播放速度倍数（默认：1.0）
sudo spank --speed 0.7   # 更慢更深沉
sudo spank --speed 1.5   # 更快
sudo spank --sexy --speed 0.6
```

### 模式

**疼痛模式**（默认）：检测到拍打时随机播放 10 个疼痛的音频片段之一。

**性感模式**（`--sexy`）：监听在 5 分钟滚动窗口内的拍打次数。拍打越多，音频回应越强烈。60 个升级级别。

**光环模式**（`--halo`）：检测到拍打时随机播放光环视频游戏系列的死亡音效。

**自定义模式**（`--custom`）：从你指定的自定义目录中随机播放 MP3 文件。

### 检测调优

使用 `--fast` 获得更响应的配置，具有更快的轮询（4ms vs 10ms）、更短的冷却时间（350ms vs 750ms）、更高的灵敏度（0.18 vs 0.05 阈值）和更大的样本批次（320 vs 200）。

需要时，你仍然可以使用 `--min-amplitude` 和 `--cooldown` 覆盖单个值。

### 灵敏度

使用 `--min-amplitude` 控制检测灵敏度（默认：`0.05`）：

- 较低值（例如 0.05-0.10）：非常敏感，检测轻拍
- 中等值（例如 0.15-0.30）：平衡的灵敏度
- 较高值（例如 0.30-0.50）：只有强烈的撞击才会触发声音

该值表示触发声音所需的最小加速度幅度（以 g 为单位）。

## 作为服务运行

要让 spank 在启动时自动运行，请创建一个`系统守护进程`配置文件。选择运行模式，如下：

<details>
<summary>疼痛模式（默认）</summary>

```bash
sudo tee /Library/LaunchDaemons/com.taigrr.spank.plist > /dev/null << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.taigrr.spank</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/spank</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/spank.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/spank.err</string>
</dict>
</plist>
EOF
```

</details>

<details>
<summary>性感模式</summary>

```bash
sudo tee /Library/LaunchDaemons/com.taigrr.spank.plist > /dev/null << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.taigrr.spank</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/spank</string>
        <string>--sexy</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/spank.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/spank.err</string>
</dict>
</plist>
EOF
```

</details>

<details>
<summary>光环模式</summary>

```bash
sudo tee /Library/LaunchDaemons/com.taigrr.spank.plist > /dev/null << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.taigrr.spank</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/spank</string>
        <string>--halo</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/spank.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/spank.err</string>
</dict>
</plist>
EOF
```

</details>

> **注意：** 如果你将 spank 安装在其他位置（例如 `~/go/bin/spank`），请更新 spank 的路径。

加载并启动服务：

```bash
sudo launchctl load /Library/LaunchDaemons/com.taigrr.spank.plist
```

由于 `plist` 文件位于 `/Library/LaunchDaemons` 且未设置 `UserName` ，`launchctl` 命令会以 root 身份运行它， 所以不需要加 `sudo`。

要停止或卸载：

```bash
sudo launchctl unload /Library/LaunchDaemons/com.taigrr.spank.plist
```

## 工作原理

1. 通过 Apple Silicon 芯片的加速度传感器直接读取原始加速度传感器数据
2. 运行振动检测（瞬态/长时瞬态、累积偏差、峰度、峰值/平均绝对偏差）
3. 当检测到显著撞击时，播放嵌入的 MP3 回应
4. **可选音量缩放**（`--volume-scaling`）— 轻拍时安静播放，重拍时以全音量播放
5. **可选速度控制**（`--speed`）— 调整播放速度和音调（0.5 = 半速，2.0 = 2倍速）
6. 有 750ms 响应冷却时间以防止快速连续播放，可通过 `--cooldown` 调整

## Star History

[![Star History Chart](https://star-history.dera.page/svg?repos=taigrr/spank&type=date&legend=top-left)](https://star-history.dera.page/#taigrr/spank&type=date&legend=top-left)

## 致谢

传感器读取和振动检测来源于 [olvvier/apple-silicon-accelerometer](https://github.com/olvvier/apple-silicon-accelerometer)。

## 许可证

MIT

<!-- Links -->
[readme-en-link]: ./README.md
