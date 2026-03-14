package main

import (
	"encoding/json"
	"fmt"
)

var scenes = map[string]int{
	"暖白": 2700,
	"睡前": 2700,
	"自然": 4000,
	"冷白": 5100,
	"阅读": 5100,
}

type Lamp struct {
	miio *MiIO
}

func NewLamp(ip, token string) (*Lamp, error) {
	m, err := NewMiIO(ip, token)
	if err != nil {
		return nil, err
	}
	return &Lamp{miio: m}, nil
}

type prop struct {
	SIID  int `json:"siid"`
	PIID  int `json:"piid"`
	Value any `json:"value,omitempty"`
}

func (l *Lamp) setProps(props []prop) error {
	_, err := l.miio.Send("set_properties", props)
	return err
}

func (l *Lamp) TurnOn() error {
	return l.setProps([]prop{{SIID: 2, PIID: 1, Value: true}})
}

func (l *Lamp) TurnOff() error {
	return l.setProps([]prop{{SIID: 2, PIID: 1, Value: false}})
}

func (l *Lamp) SetBrightness(v int) error {
	if v < 1 {
		v = 1
	} else if v > 100 {
		v = 100
	}
	return l.setProps([]prop{{SIID: 2, PIID: 2, Value: v}})
}

func (l *Lamp) SetColorTemp(k int) error {
	if k < 2700 {
		k = 2700
	} else if k > 5100 {
		k = 5100
	}
	return l.setProps([]prop{{SIID: 2, PIID: 3, Value: k}})
}

func (l *Lamp) SetScene(name string, brightness *int) error {
	ct, ok := scenes[name]
	if !ok {
		return fmt.Errorf("未知场景: %s（可用：暖白/自然/冷白/阅读/睡前）", name)
	}
	props := []prop{{SIID: 2, PIID: 3, Value: ct}}
	if brightness != nil {
		v := *brightness
		if v < 1 {
			v = 1
		} else if v > 100 {
			v = 100
		}
		props = append(props, prop{SIID: 2, PIID: 2, Value: v})
	}
	return l.setProps(props)
}

type Status struct {
	On         bool
	Brightness int
	ColorTemp  int
}

func (l *Lamp) GetStatus() (*Status, error) {
	result, err := l.miio.Send("get_properties", []prop{
		{SIID: 2, PIID: 1},
		{SIID: 2, PIID: 2},
		{SIID: 2, PIID: 3},
	})
	if err != nil {
		return nil, err
	}

	var props []struct {
		SIID  int             `json:"siid"`
		PIID  int             `json:"piid"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(result, &props); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}
	if len(props) < 3 {
		return nil, fmt.Errorf("unexpected status response")
	}

	var on bool
	var brightness, colorTemp int
	json.Unmarshal(props[0].Value, &on)
	json.Unmarshal(props[1].Value, &brightness)
	json.Unmarshal(props[2].Value, &colorTemp)

	return &Status{On: on, Brightness: brightness, ColorTemp: colorTemp}, nil
}
