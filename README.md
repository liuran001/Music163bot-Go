# Music163bot-Go v3

一个用于下载/分享网易云音乐的 Telegram Bot（重构版）。

原始项目：https://github.com/XiaoMengXinX/Music163bot-Go

## 依赖

- Go 1.23+
- ffmpeg（/recognize 语音识别需要）

## 配置

复制 `config_example.ini` 为 `config.ini` 并填写：

```ini
BOT_TOKEN = YOUR_BOT_TOKEN
MUSIC_U = YOUR_MUSIC_U
```

## 运行

```bash
go build
./Music163bot-Go -c config.ini
```

## 架构

详见 `ARCHITECTURE.md`。

## 许可证

GPL-3.0
