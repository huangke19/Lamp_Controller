#!/usr/bin/env python3
import os
import logging
from telegram import Update
from telegram.ext import Application, CommandHandler, ContextTypes

# Simple .env loader
def load_dotenv(path=".env"):
    try:
        with open(path, "r") as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith("#"):
                    key, value = line.split("=", 1)
                    os.environ[key] = value
    except FileNotFoundError:
        pass

load_dotenv()

# Enable logging
logging.basicConfig(
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s", level=logging.INFO
)
logger = logging.getLogger(__name__)

# Import lamp module
import lamp

async def lamp_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    args = context.args

    if not args:
        await update.message.reply_text(
            "用法：\n"
            "/lamp 开\n"
            "/lamp 关\n"
            "/lamp 状态\n"
            "/lamp 暖白 [亮度1-100]\n"
            "/lamp 自然 [亮度1-100]\n"
            "/lamp 冷白 [亮度1-100]\n"
            "/lamp 阅读 [亮度1-100]\n"
            "/lamp 睡前 [亮度1-100]"
        )
        return

    try:
        cmd = args[0]
        brightness = int(args[1]) if len(args) > 1 else None

        if cmd == "开":
            lamp.turn_on()
            await update.message.reply_text("台灯已开")

        elif cmd == "关":
            lamp.turn_off()
            await update.message.reply_text("台灯已关")

        elif cmd == "状态":
            s = lamp.get_status()
            on_str = "开" if s["on"] else "关"
            await update.message.reply_text(
                f"状态：{on_str}\n亮度：{s['brightness']}%\n色温：{s['color_temp']}K"
            )

        else:
            lamp.set_scene(cmd, brightness)
            msg = f"场景：{cmd}"
            if brightness:
                msg += f"，亮度：{brightness}%"
            await update.message.reply_text(msg)

    except ValueError as e:
        await update.message.reply_text(f"参数错误：{e}")
    except Exception as e:
        await update.message.reply_text(f"台灯控制失败：{e}")

async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Send a message when the command /start is issued."""
    await update.message.reply_text('Hi! Use /lamp to control the lamp.')

async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Send a message when the command /help is issued."""
    await update.message.reply_text('Help!')

def main():
    """Start the bot."""
    # Create the Application and pass it your bot's token.
    token = os.getenv("TELEGRAM_BOT_TOKEN")
    if not token:
        logger.error("TELEGRAM_BOT_TOKEN not set in .env")
        return

    application = Application.builder().token(token).build()

    # on different commands - answer in Telegram
    application.add_handler(CommandHandler("start", start))
    application.add_handler(CommandHandler("help", help_command))
    application.add_handler(CommandHandler("lamp", lamp_handler))

    # Run the bot until the user presses Ctrl-C
    application.run_polling(allowed_updates=Update.ALL_TYPES)

if __name__ == "__main__":
    main()