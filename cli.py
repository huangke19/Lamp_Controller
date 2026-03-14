#!/usr/bin/env python3
"""命令行控制台灯，单次执行，用完即退。"""

import os
import sys

from dotenv import load_dotenv

load_dotenv()

SCENES = {"暖白", "自然", "冷白", "阅读", "睡前"}

HELP_TEXT = """\
用法：
  python cli.py 开
  python cli.py 关
  python cli.py 状态
  python cli.py 亮度 <1-100>
  python cli.py 色温 <2700-5100>
  python cli.py <场景> [亮度1-100]

场景：暖白 / 自然 / 冷白 / 阅读 / 睡前"""


def check_env():
    missing = [k for k in ("LAMP_IP", "LAMP_TOKEN") if not os.getenv(k)]
    if missing:
        print(f"错误：.env 文件中缺少配置项：{', '.join(missing)}", file=sys.stderr)
        sys.exit(1)


def main():
    check_env()

    import lamp  # 延迟导入，确保环境变量已就绪

    args = sys.argv[1:]

    if not args:
        print(HELP_TEXT)
        sys.exit(0)

    cmd = args[0]

    try:
        if cmd == "开":
            lamp.turn_on()
            print("台灯已开")

        elif cmd == "关":
            lamp.turn_off()
            print("台灯已关")

        elif cmd == "状态":
            s = lamp.get_status()
            on_str = "开" if s["on"] else "关"
            print(f"状态：{on_str}\n亮度：{s['brightness']}%\n色温：{s['color_temp']}K")

        elif cmd == "亮度":
            if len(args) < 2:
                print("错误：请指定亮度值，例如：python cli.py 亮度 80", file=sys.stderr)
                sys.exit(1)
            value = int(args[1])
            lamp.set_brightness(value)
            print(f"亮度已设置：{max(1, min(100, value))}%")

        elif cmd == "色温":
            if len(args) < 2:
                print("错误：请指定色温值，例如：python cli.py 色温 4000", file=sys.stderr)
                sys.exit(1)
            kelvin = int(args[1])
            lamp.set_color_temp(kelvin)
            print(f"色温已设置：{max(2700, min(5100, kelvin))}K")

        elif cmd in SCENES:
            brightness = int(args[1]) if len(args) > 1 else None
            lamp.set_scene(cmd, brightness)
            msg = f"场景：{cmd}"
            if brightness is not None:
                msg += f"，亮度：{brightness}%"
            print(msg)

        else:
            print(f"错误：未知命令 '{cmd}'\n", file=sys.stderr)
            print(HELP_TEXT, file=sys.stderr)
            sys.exit(1)

    except ValueError as e:
        print(f"错误：参数无效 —— {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"台灯控制失败：{e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
