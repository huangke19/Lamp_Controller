// device.go 封装台灯的控制操作，是最高层的业务逻辑层。
// 它依赖 miio.go 提供的底层通信，对外提供语义清晰的方法（开灯、关灯等）。
package main

import (
	"encoding/json" // JSON 解析，用于读取设备状态响应
	"fmt"           // 格式化错误信息
)

// scenePreset 是场景预设（色温 + 默认亮度）。
//
// 预设值按照照明语义做区分：
//   - 暖白：偏暖、舒适，常规亮度
//   - 自然：中性白，均衡亮度
//   - 冷白：偏冷、清晰，较高亮度
//   - 阅读：中偏冷，确保对比和舒适度
//   - 睡前：暖光低亮，减少刺激
type scenePreset struct {
	ColorTemp  int
	Brightness int
}

var scenes = map[string]scenePreset{
	"warm":    {ColorTemp: 3000, Brightness: 70},
	"neutral": {ColorTemp: 4000, Brightness: 65},
	"cool":    {ColorTemp: 5000, Brightness: 85},
	"focus":   {ColorTemp: 4500, Brightness: 80},
	"night":   {ColorTemp: 2700, Brightness: 20},
}

var sceneAliases = map[string]string{
	"warm":    "warm",
	"暖白":      "warm",
	"neutral": "neutral",
	"自然":      "neutral",
	"cool":    "cool",
	"冷白":      "cool",
	"focus":   "focus",
	"阅读":      "focus",
	"night":   "night",
	"睡前":      "night",
}

func normalizeSceneName(name string) (string, bool) {
	scene, ok := sceneAliases[name]
	return scene, ok
}

// Lamp 是台灯的控制结构体，持有一个 MiIO 通信对象。
//
// 这是一种"组合"模式：Lamp 包含 MiIO，而不是继承它。
// Go 没有继承，用组合代替（类似 Python 的组合模式）。
type Lamp struct {
	miio *MiIO // *MiIO 是指向 MiIO 结构体的指针（小写 miio 表示私有字段）
}

// NewLamp 创建一个台灯控制对象。
//
// 这是工厂函数模式（等价于 Python 的 __init__）。
// 先创建 MiIO 底层通信对象，再包装成 Lamp。
func NewLamp(ip, token string) (*Lamp, error) {
	m, err := NewMiIO(ip, token)
	if err != nil {
		// 把错误向上传递，不在这里处理。
		// 等价于 Python 的 raise（但不是异常，只是返回错误值）。
		return nil, err
	}
	// &Lamp{miio: m} 创建 Lamp 结构体并取地址，返回指针。
	return &Lamp{miio: m}, nil
}

// prop 是 MiOT 协议中的属性结构体。
//
// 小米设备用 siid（服务 ID）和 piid（属性 ID）来标识每个可控属性：
//
//	siid=2, piid=1 → 开关
//	siid=2, piid=2 → 亮度（1-100）
//	siid=2, piid=3 → 色温（2700-5100K）
//
// 字段标签 `json:"siid"` 控制 JSON 序列化时的字段名（小写）。
// omitempty 表示当 Value 为零值时不输出这个字段（查询状态时不需要 value）。
type prop struct {
	SIID  int `json:"siid"`
	PIID  int `json:"piid"`
	Value any `json:"value,omitempty"` // any 表示任意类型（bool、int 等）
}

// setProps 是内部辅助方法，发送 set_properties 命令。
//
// 方法名首字母小写（setProps），表示这是私有方法，只能在包内调用。
// 等价于 Python 的 def _set_props(self, ...)。
// []prop 是 prop 切片，等价于 Python 的 list[prop]。
func (l *Lamp) setProps(props []prop) error {
	// l.miio.Send 调用 MiIO 的 Send 方法发送命令。
	// _ 忽略返回的 json.RawMessage（set 命令的响应不需要解析）。
	_, err := l.miio.Send("set_properties", props)
	return err
}

// TurnOn 开灯。
//
// 发送 set_properties 命令，把 siid=2 piid=1（开关属性）设为 true。
// 方法名首字母大写（TurnOn）表示公开方法。
func (l *Lamp) TurnOn() error {
	// []prop{{...}} 创建一个只有一个元素的 prop 切片。
	// 等价于 Python 的 [prop(siid=2, piid=1, value=True)]。
	// Go 里结构体字面量不用写字段名（如果按顺序）也可以，
	// 但写字段名更清晰，推荐这种方式。
	return l.setProps([]prop{{SIID: 2, PIID: 1, Value: true}})
}

// TurnOff 关灯。
func (l *Lamp) TurnOff() error {
	return l.setProps([]prop{{SIID: 2, PIID: 1, Value: false}})
}

// SetBrightness 设置亮度，范围 1-100。
// 超出范围的值会自动截断到边界。
func (l *Lamp) SetBrightness(v int) error {
	// Go 没有 clamp 函数，只能用 if/else 手动截断。
	if v < 1 {
		v = 1
	} else if v > 100 {
		v = 100
	}
	return l.setProps([]prop{{SIID: 2, PIID: 2, Value: v}})
}

// SetColorTemp 设置色温，范围 2700-5100K。
func (l *Lamp) SetColorTemp(k int) error {
	if k < 2700 {
		k = 2700
	} else if k > 5100 {
		k = 5100
	}
	return l.setProps([]prop{{SIID: 2, PIID: 3, Value: k}})
}

// SetScene 设置场景模式（预设色温 + 默认亮度，可选亮度覆盖默认值）。
//
// brightness 是 *int（指向整数的指针），而不是 int。
// 这是 Go 表达"可选参数"的惯用方式：
//   - brightness == nil → 没有传亮度，保持当前亮度
//   - brightness != nil → 有传亮度，*brightness 取出实际值
//
// Python 的等价写法是 brightness: Optional[int] = None。
func (l *Lamp) SetScene(name string, brightness *int) error {
	name, ok := normalizeSceneName(name)
	if !ok {
		return fmt.Errorf("unknown scene: %s (available: warm/neutral/cool/focus/night)", name)
	}

	// 从 scenes map 查找预设值。
	// map 查找返回两个值：value 和 ok（是否找到）。
	preset := scenes[name]

	// 默认使用场景预设亮度；如果调用方传了亮度，则覆盖。
	v := preset.Brightness
	if brightness != nil {
		v = *brightness
	}
	if v < 1 {
		v = 1
	} else if v > 100 {
		v = 100
	}

	// 场景一次请求同时设置色温和亮度。
	props := []prop{
		{SIID: 2, PIID: 3, Value: preset.ColorTemp},
		{SIID: 2, PIID: 2, Value: v},
	}

	return l.setProps(props)
}

// Status 存储台灯当前状态的结构体。
type Status struct {
	On         bool // 是否开灯
	Brightness int  // 亮度，1-100
	ColorTemp  int  // 色温，2700-5100K
}

// GetStatus 查询台灯当前状态，返回 *Status 指针和 error。
func (l *Lamp) GetStatus() (*Status, error) {
	// 发送 get_properties 查询三个属性。
	// 查询时只需要 SIID 和 PIID，不需要 Value（omitempty 会自动省略）。
	result, err := l.miio.Send("get_properties", []prop{
		{SIID: 2, PIID: 1}, // 开关状态
		{SIID: 2, PIID: 2}, // 亮度
		{SIID: 2, PIID: 3}, // 色温
	})
	if err != nil {
		return nil, err
	}

	// 设备返回一个 JSON 数组，每个元素是一个属性值对象。
	// 格式：[{"siid":2,"piid":1,"value":true}, {"siid":2,"piid":2,"value":80}, ...]
	//
	// 这里用匿名结构体切片来解析响应。
	// []struct{...} 是匿名结构体的切片，不需要提前命名这个类型。
	// json.RawMessage 表示先不解析 value 字段，因为它的类型不固定（bool 或 int）。
	var props []struct {
		SIID  int             `json:"siid"`
		PIID  int             `json:"piid"`
		Value json.RawMessage `json:"value"` // 延迟解析，稍后按实际类型解析
	}

	// json.Unmarshal 把 result（[]byte）解析到 props 切片。
	// &props 传指针，让 Unmarshal 能写入数据。
	if err := json.Unmarshal(result, &props); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}
	if len(props) < 3 {
		return nil, fmt.Errorf("unexpected status response")
	}

	// 分别解析三个属性的值到对应类型。
	// var on bool 声明变量，Go 会自动初始化为零值（false）。
	var on bool
	var brightness, colorTemp int

	// 按 siid/piid 匹配属性值，不依赖返回顺序。
	found := 0
	for _, p := range props {
		if p.SIID != 2 {
			continue
		}
		switch p.PIID {
		case 1:
			if err := json.Unmarshal(p.Value, &on); err != nil {
				return nil, fmt.Errorf("failed to parse on state: %w", err)
			}
			found++
		case 2:
			if err := json.Unmarshal(p.Value, &brightness); err != nil {
				return nil, fmt.Errorf("failed to parse brightness: %w", err)
			}
			found++
		case 3:
			if err := json.Unmarshal(p.Value, &colorTemp); err != nil {
				return nil, fmt.Errorf("failed to parse color temp: %w", err)
			}
			found++
		}
	}
	if found < 3 {
		return nil, fmt.Errorf("incomplete status response: expected 3 properties, got %d", found)
	}

	return &Status{On: on, Brightness: brightness, ColorTemp: colorTemp}, nil
}
