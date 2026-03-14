import os
from miio import MiotDevice


def _get_lamp():
    ip = os.getenv("LAMP_IP")
    token = os.getenv("LAMP_TOKEN")
    return MiotDevice(ip=ip, token=token)


def turn_on():
    _get_lamp().send('set_properties', [{'siid': 2, 'piid': 1, 'value': True}])


def turn_off():
    _get_lamp().send('set_properties', [{'siid': 2, 'piid': 1, 'value': False}])


def set_brightness(value: int):
    """value: 1–100"""
    value = max(1, min(100, value))
    _get_lamp().send('set_properties', [{'siid': 2, 'piid': 2, 'value': value}])


def set_color_temp(kelvin: int):
    """kelvin: 2700–5100"""
    kelvin = max(2700, min(5100, kelvin))
    _get_lamp().send('set_properties', [{'siid': 2, 'piid': 3, 'value': kelvin}])


def set_scene(name: str, brightness: int = None):
    """name: 暖白 / 自然 / 冷白 / 阅读 / 睡前"""
    presets = {
        "暖白": 2700,
        "睡前": 2700,
        "自然": 4000,
        "冷白": 5100,
        "阅读": 5100,
    }
    ct = presets.get(name)
    if ct is None:
        raise ValueError(f"未知场景: {name}")
    lamp = _get_lamp()
    props = [{'siid': 2, 'piid': 3, 'value': ct}]
    if brightness is not None:
        brightness = max(1, min(100, brightness))
        props.append({'siid': 2, 'piid': 2, 'value': brightness})
    lamp.send('set_properties', props)


def get_status() -> dict:
    result = _get_lamp().send('get_properties', [
        {'siid': 2, 'piid': 1},
        {'siid': 2, 'piid': 2},
        {'siid': 2, 'piid': 3},
    ])
    return {
        'on': result[0]['value'],
        'brightness': result[1]['value'],
        'color_temp': result[2]['value'],
    }