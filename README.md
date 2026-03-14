# OpenClaw 台灯控制模块

小米台灯2（xiaomi.light.lamp31）的本地控制模块，集成到 OpenClaw Telegram Bot。

## 📋 功能特性

- ✅ **本地控制**：通过局域网 UDP 直接控制，无需云端
- ✅ **完整功能**：开关、亮度（1-100%）、色温（2700-5100K）
- ✅ **场景模式**：暖白、自然、冷白、阅读、睡前五种预设
- ✅ **状态查询**：实时获取台灯状态
- ✅ **命令行控制**：`lamp 开` 即可，Go 编译单二进制，无需 Python 环境
- ✅ **Telegram Bot**：通过 `/lamp` 命令远程控制
- ✅ **安全配置**：敏感信息存储在 `.env` 文件中

## 🛠️ 硬件要求

- 小米台灯2（型号：`xiaomi.light.lamp31`）
- 局域网环境（设备与运行 Bot 的电脑在同一网络）
- 设备 IP：`192.168.5.40`（需根据实际网络调整）
- 设备 Token：32位十六进制字符串

## 📦 软件依赖

### 命令行控制（Go）
- Go 1.18+（仅编译时需要，运行时零依赖）

### Telegram Bot（Python）
```txt
python-miio>=0.5.12
python-telegram-bot[job-queue]>=20.7
python-dotenv>=1.0.0
```

### 可选依赖（用于 token 提取）
- `requests` - HTTP 请求
- `colorama` - 终端颜色
- `Pillow` - 图像处理
- `pycryptodome` / `pycrypto` - 加密算法

## 🚀 快速开始

### 1. 克隆/下载项目
```bash
git clone <repository-url>
cd mylamp
```

### 2. 安装依赖
```bash
pip install -r requirements.txt
```

### 3. 配置设备信息
创建或编辑 `.env` 文件：

```env
# 设备配置
LAMP_IP=192.168.x.x
LAMP_TOKEN=your_32_char_hex_token_here

# Telegram Bot 配置
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here
```

**重要**：请替换为您的实际设备信息：
- `LAMP_IP`：台灯在局域网中的 IP 地址
- `LAMP_TOKEN`：设备的 32 位 token（可使用 `token_extractor.py` 提取）
- `TELEGRAM_BOT_TOKEN`：Telegram Bot Token（通过 @BotFather 创建）

### 4. 编译命令行工具

```bash
cd lamp
go build -o ~/bin/lamp .
```

编译成功后，任意目录可直接运行：

```bash
lamp 状态
lamp 开
```

### 5. 验证配置（可选）
```bash
python test_lamp.py
```
预期输出：
```
LAMP_IP: 192.168.5.40
LAMP_TOKEN: d208a096...
Lamp module imported successfully.
Functions available:
  lamp.turn_on()
  lamp.turn_off()
  lamp.set_brightness(value)
  lamp.set_color_temp(kelvin)
  lamp.set_scene(name, brightness)
  lamp.get_status()
Current status: {'on': True, 'brightness': 50, 'color_temp': 4000}
```

### 6. 启动 Bot（可选）
```bash
python bot.py
```
正常启动后显示：
```
2026-03-14 21:06:27,108 - telegram.ext.Application - INFO - Application started
```

## 💻 命令行控制

无需维持进程，每次执行一条命令即退出：

```bash
python cli.py 开
python cli.py 关
python cli.py 状态
python cli.py 亮度 80
python cli.py 色温 4000
python cli.py 自然
python cli.py 冷白 80
```

### 支持的命令

| 命令 | 参数 | 说明 |
|------|------|------|
| `开` | 无 | 开灯 |
| `关` | 无 | 关灯 |
| `状态` | 无 | 查询当前状态 |
| `亮度` | 1-100 | 直接设置亮度 |
| `色温` | 2700-5100 | 直接设置色温（K） |
| `暖白` | 可选亮度 1-100 | 2700K 暖白光 |
| `自然` | 可选亮度 1-100 | 4000K 自然光 |
| `冷白` | 可选亮度 1-100 | 5100K 冷白光 |
| `阅读` | 可选亮度 1-100 | 5100K 阅读模式 |
| `睡前` | 可选亮度 1-100 | 2700K 睡前模式 |

不带参数运行显示帮助：

```bash
python cli.py
```

## 📱 Telegram 命令

### 基本命令
| 命令 | 参数 | 说明 |
|------|------|------|
| `/lamp 开` | 无 | 开灯 |
| `/lamp 关` | 无 | 关灯 |
| `/lamp 状态` | 无 | 查询状态 |
| `/lamp` | 无 | 显示帮助 |

### 场景模式命令
| 命令 | 可选亮度 | 说明 |
|------|----------|------|
| `/lamp 暖白` | 1-100 | 2700K 暖白光 |
| `/lamp 自然` | 1-100 | 4000K 自然光 |
| `/lamp 冷白` | 1-100 | 5100K 冷白光 |
| `/lamp 阅读` | 1-100 | 5100K 阅读模式 |
| `/lamp 睡前` | 1-100 | 2700K 睡前模式 |

**示例**：
- `/lamp 自然` - 自然光模式，默认亮度
- `/lamp 冷白 80` - 冷白模式，亮度 80%

## 🔧 模块 API

### 导入模块
```python
import lamp
```

### 可用函数
```python
# 开关控制
lamp.turn_on()        # 开灯
lamp.turn_off()       # 关灯

# 亮度控制
lamp.set_brightness(75)  # 亮度 75%（范围：1-100）

# 色温控制
lamp.set_color_temp(4000)  # 色温 4000K（范围：2700-5100）

# 场景模式
lamp.set_scene("自然", 80)  # 自然光，亮度 80%
lamp.set_scene("暖白")      # 暖白光，默认亮度

# 状态查询
status = lamp.get_status()
# 返回：{'on': True, 'brightness': 50, 'color_temp': 4000}
```

## 🗂️ 项目结构

```
mylamp/
├── .env                    # 环境配置文件（敏感信息，不提交 Git）
├── .env.example            # 配置模板
├── bot.py                 # Telegram Bot 主程序
├── cli.py                 # 命令行控制入口（Python 备用版）
├── lamp/                  # Go CLI 源码
│   ├── go.mod
│   ├── main.go            # CLI 入口
│   ├── miio.go            # MiIO 协议层（UDP + AES）
│   └── device.go          # 台灯控制层
├── lamp.py                # 台灯控制核心模块（供 bot.py 使用）
├── test_lamp.py           # 测试脚本
├── token_extractor.py     # Token 提取工具
├── requirements.txt       # Python 依赖
└── README.md             # 本说明文档
```

## 🔍 设备 Token 获取

如果不知道设备 token，可使用内置工具提取：

```bash
python token_extractor.py
```

按提示操作：
1. 选择登录方式（密码或二维码）
2. 选择服务器区域（如 `cn`）
3. 查找设备 "台灯" 对应的 token

## ⚠️ 故障排除

### 1. Bot 无法启动
**错误**：`Conflict: terminated by other getUpdates request`
**解决方案**：
```bash
# 检查并停止其他 Bot 进程
ps aux | grep -E "python.*bot\.py|telegram"
kill <进程ID>
```

### 2. 设备连接失败
**错误**：`台灯控制失败：...`
**检查步骤**：
1. 验证 IP 是否可达：`ping 192.168.5.40`
2. 确认 token 正确：`python test_lamp.py`
3. 检查网络连接：设备与电脑在同一局域网

### 3. 权限问题
**错误**：`Permission denied` 或 `ModuleNotFoundError`
**解决方案**：
```bash
# 确保依赖已安装
pip install -r requirements.txt

# 或使用虚拟环境
python -m venv myenv
source myenv/bin/activate
pip install -r requirements.txt
```

## 🔒 安全注意事项

1. **保护敏感信息**：
   - `.env` 文件包含设备 token 和 Bot token
   - **切勿提交到 Git**：在 `.gitignore` 中添加 `.env`
   - 建议将 `.env.example` 作为模板提交

2. **网络安全**：
   - 设备在局域网内通信，相对安全
   - 可考虑设置防火墙规则限制访问

3. **Bot 安全**：
   - 限制 Bot 访问权限
   - 定期更新 token

## 📝 开发说明

### 技术细节
- **通信协议**：MiOT over UDP port 54321
- **加密**：AES-128-CBC，key = MD5(token)，iv = MD5(key + token)
- **CLI 实现**：Go 标准库（`crypto/aes`、`crypto/md5`、`net`），零外部依赖
- **Bot 实现**：python-miio + python-telegram-bot
- **参数映射**：
  - 开关：`siid=2, piid=1`
  - 亮度：`siid=2, piid=2`（1-100）
  - 色温：`siid=2, piid=3`（2700-5100K）

### 扩展开发
如需支持其他小米设备，可参考 `lamp.py` 的模式：
1. 确定设备的 `siid/piid` 映射
2. 实现相应的控制函数
3. 集成到 Bot 命令处理器

## 📄 许可证

本项目遵循 MIT 许可证。详见 LICENSE 文件（如有）。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request。

## 📞 支持

如有问题，请：
1. 查看本文档的故障排除部分
2. 提交 Issue 到项目仓库
3. 联系维护者

---

**最后更新**：2026-03-14
**版本**：1.0.0
**维护者**：OpenClaw 项目组