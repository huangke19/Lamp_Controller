#!/usr/bin/env python3
import os
import sys

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

# Test lamp module
if __name__ == "__main__":
    # Check environment variables
    ip = os.getenv("LAMP_IP")
    token = os.getenv("LAMP_TOKEN")

    if not ip or not token:
        print("Please set LAMP_IP and LAMP_TOKEN in .env file")
        print("Example .env content:")
        print("LAMP_IP=192.168.5.40")
        print("LAMP_TOKEN=你的32位token")
        sys.exit(1)

    print(f"LAMP_IP: {ip}")
    print(f"LAMP_TOKEN: {token[:8]}...")

    # Test lamp functions (commented out to avoid accidental control)
    print("\nLamp module imported successfully.")
    print("Functions available:")
    import lamp
    print("  lamp.turn_on()")
    print("  lamp.turn_off()")
    print("  lamp.set_brightness(value)")
    print("  lamp.set_color_temp(kelvin)")
    print("  lamp.set_scene(name, brightness)")
    print("  lamp.get_status()")

    #Example usage (uncomment to test):
    status = lamp.get_status()
    print(f"Current status: {status}")