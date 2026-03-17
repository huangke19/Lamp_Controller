# 台灯控制

本项目是一个用于 **小米台灯2（`xiaomi.light.lamp31`）** 的本地控制工具。
通过局域网 UDP 直接通信，不依赖云端。

支持两种使用方式：
- 命令行（CLI）
- Web 控制台（含 iOS 可安装 PWA）

---

## 功能特性

- 本地直连控制（MiOT over UDP）
- 开关、亮度、色温控制
- 场景模式（预设色温 + 预设亮度）
- 状态查询
- Web 控制台：
  - 实时拖动亮度/色温
  - 手动输入数值（回车提交）
  - 浅色 / 深色双主题切换（自动跟随系统，可手动记忆）
  - iPhone 端单屏全屏布局（含安全区适配）
- PWA 支持（可添加到 iOS 主屏幕）
- 单二进制运行（Go 编译，无额外运行时依赖）

---

## 快速开始

### 1. 配置设备信息

创建 `.env`（可参考 `.env.example`）：

```env
LAMP_IP=192.168.x.x
LAMP_TOKEN=你的32位十六进制token
```

说明：
- `LAMP_IP`：台灯在局域网中的 IP
- `LAMP_TOKEN`：设备 token（32 位十六进制）

可选配置：

```env
LAMP_DEBUG=1
LAMP_WEB_ADDR=:8080
```

### 2. 编译

```bash
cd lamp
go build -o ~/bin/lamp .
```

确保 `~/bin` 在 `PATH` 中。

### 3. 运行示例

```bash
lamp 开
lamp 状态
lamp 睡前
lamp 阅读 70
```

---

## 命令参考

### 总览

| 命令 | 语法 | 说明 |
|---|---|---|
| 开灯 | `lamp 开` | 打开台灯 |
| 关灯 | `lamp 关` | 关闭台灯 |
| 查询状态 | `lamp 状态` | 返回开关、亮度、色温 |
| 设置亮度 | `lamp 亮度 <1-100>` | 超范围会自动截断到 `1-100` |
| 设置色温 | `lamp 色温 <2700-5100>` | 超范围会自动截断到 `2700-5100` |
| 场景模式 | `lamp <场景> [亮度]` | 场景默认“色温+亮度”，可选亮度仅覆盖亮度 |
| Web 控制台 | `lamp serve [监听地址]` | 启动 Web 服务（默认 `:8080`） |

### 基础控制命令

1. 开灯

```bash
lamp 开
```

2. 关灯

```bash
lamp 关
```

3. 查询状态

```bash
lamp 状态
```

返回示例：

```text
状态：开
亮度：60%
色温：4000K
```

### 数值控制命令

1. 设置亮度

```bash
lamp 亮度 80
```

边界行为：
- 小于 `1` 会被截断成 `1`
- 大于 `100` 会被截断成 `100`

示例：

```bash
lamp 亮度 0     # 实际生效为 1
lamp 亮度 150   # 实际生效为 100
```

2. 设置色温

```bash
lamp 色温 4200
```

边界行为：
- 小于 `2700` 会被截断成 `2700`
- 大于 `5100` 会被截断成 `5100`

示例：

```bash
lamp 色温 2600  # 实际生效为 2700
lamp 色温 6000  # 实际生效为 5100
```

### 场景命令（预设）

语法：

```bash
lamp <场景名> [亮度]
```

场景预设如下：

| 场景 | 默认色温 | 默认亮度 |
|---|---:|---:|
| 暖白 | 3000K | 70% |
| 自然 | 4000K | 65% |
| 冷白 | 5000K | 85% |
| 阅读 | 4500K | 80% |
| 睡前 | 2700K | 20% |

规则：
- 不带亮度参数：应用该场景的“默认色温 + 默认亮度”
- 带亮度参数：使用场景默认色温，亮度用你传入值（并按 `1-100` 截断）

示例：

```bash
lamp 阅读        # 4500K + 80%
lamp 睡前        # 2700K + 20%
lamp 自然 40     # 4000K + 40%
lamp 冷白 120    # 5000K + 100%（截断）
```

### Web 服务命令

1. 默认启动

```bash
lamp serve
```

2. 指定监听地址

```bash
lamp serve 0.0.0.0:8080
```

3. 用环境变量指定地址

```bash
LAMP_WEB_ADDR=:8080 lamp serve
```

提示：
- 本地访问：`http://127.0.0.1:8080`
- 同局域网手机访问：`http://<你的电脑IP>:8080`

### 帮助与错误

1. 不带参数运行会显示帮助

```bash
lamp
```

2. 参数不是整数时会报错并退出（退出码 1）

```bash
lamp 亮度 abc
lamp 色温 foo
```

---

## 调试模式

开启调试日志（握手、请求参数、响应）：

```bash
LAMP_DEBUG=1 lamp 状态
```

或：

```bash
LAMP_DEBUG=1 lamp 睡前 30
```

---

## Web 控制台

启动：

```bash
lamp serve
```

默认地址：
- `http://127.0.0.1:8080`

也可以：

```bash
lamp serve 0.0.0.0:8080
```

或：

```bash
LAMP_WEB_ADDR=:8080 lamp serve
```

---

## iOS 使用（PWA）

1. iPhone Safari 打开：`http://<主机IP>:8080`
2. 点分享按钮 -> “添加到主屏幕”
3. 以后可像 App 一样从主屏幕启动

说明：
- 主题默认跟随系统浅/深色
- 页面内可手动切换主题

### 界面预览（iPhone 13）

浅色：

![浅色主题](https://raw.githubusercontent.com/huangke19/Lamp_Controller/main/docs/screenshots/mobile-light.png)

深色：

![深色主题](https://raw.githubusercontent.com/huangke19/Lamp_Controller/main/docs/screenshots/mobile-dark.png)

---

## 获取设备 Token

1. 在米家 App 将台灯接入局域网
2. 使用 [python-miio](https://github.com/rytilahti/python-miio) 的 `miiocli` 或抓包工具获取 token
3. token 为 32 位十六进制字符串

---

## 技术细节

- 协议：MiOT over UDP (`54321`)
- 加密：AES-128-CBC
  - `key = MD5(token)`
  - `iv  = MD5(key + token)`
- 参数映射：
  - 开关：`siid=2, piid=1`
  - 亮度：`siid=2, piid=2`
  - 色温：`siid=2, piid=3`
- 依赖：Go 标准库 + 前端原生 HTML/CSS/JS（无前端框架）

---

## 项目结构

```text
mylamp/
├── .env
├── .env.example
├── README.md
├── LAMP_AGENT_GUIDE.md
├── docs/
│   └── screenshots/
│       ├── mobile-light.png
│       └── mobile-dark.png
└── lamp/
    ├── go.mod
    ├── main.go
    ├── miio.go
    ├── device.go
    ├── web.go
    └── web/
        ├── index.html
        ├── manifest.webmanifest
        ├── sw.js
        ├── icon.svg
        ├── icon-180.png
        └── icon-512.png
```

---

## 安全注意事项

- `.env` 含设备 token，禁止提交到公开仓库
- 建议仅在可信局域网内使用
- 如果需要外网访问，建议通过 VPN（例如 Tailscale），不要直接暴露公网端口
