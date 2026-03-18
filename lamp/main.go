// Go 源文件必须以 package 声明开头。
// 所有文件写 "package main" 表示这是一个可执行程序（而不是库）。
// 同一目录下所有 .go 文件共享同一个 package，可以直接互相调用函数和变量，
// 不需要像 Python 一样 import 来导入同目录的文件。
package main

// import 导入标准库包。Go 的标准库非常丰富，不需要 pip install。
// 每个包名就是一个命名空间，调用时写 包名.函数名，例如 fmt.Println。
import (
	"bufio"   // 带缓冲的 I/O，用于逐行读取文件（类似 Python 的 for line in f）
	"fmt"     // 格式化输出，类似 Python 的 print / format
	"os"      // 操作系统接口：文件、环境变量、命令行参数、退出程序
	"strconv" // 字符串与数值互转，类似 Python 的 int()、str()
	"strings" // 字符串操作，类似 Python 的 str.strip()、str.split() 等
)

// const 定义常量，值在编译时确定，运行时不能修改。
// Python 里没有真正的常量，只是约定用大写变量名表示常量。
// 反引号 ` ` 是 Go 的原始字符串字面量，等价于 Python 的三引号字符串 """..."""，
// 内容原样保留，不处理转义字符（\n 就是两个字符，不是换行）。
const helpText = `用法：
  lamp 开
  lamp 关
  lamp 状态
  lamp 亮度 <1-100>
  lamp 色温 <2700-5100>
  lamp serve [监听地址]
  lamp <场景> [亮度1-100]

场景：暖白 / 自然 / 冷白 / 阅读 / 睡前`

// loadDotEnv 读取指定路径的 .env 文件，把其中的 KEY=VALUE 写入环境变量。
//
// Go 函数声明格式：func 函数名(参数名 参数类型) 返回类型
// 这个函数没有返回值（Python 中等价于没有 return 语句）。
// Go 是静态类型语言，每个参数都必须声明类型。
func loadDotEnv(path string) {
	// os.Open 打开文件，返回两个值：文件对象 f 和错误 err。
	// Go 不用 try/except，而是把错误作为返回值返回，调用方负责检查。
	// := 是"声明并赋值"的简写，等价于 Python 的直接赋值，
	// 但 Go 会自动推断变量类型（这里 f 是 *os.File，err 是 error）。
	f, err := os.Open(path)

	// 惯用的错误处理模式：if err != nil { 处理错误; return }
	// nil 是 Go 的空值，类似 Python 的 None，表示"没有错误"。
	// 文件不存在时直接返回（静默忽略），因为 .env 是可选的。
	if err != nil {
		return
	}

	// defer 让这条语句在"当前函数返回时"执行，无论是正常返回还是出错。
	// 类似 Python 的 with 语句（上下文管理器），确保文件一定会被关闭。
	// Python 写法：with open(path) as f:
	defer f.Close()

	// bufio.NewScanner 创建一个逐行扫描器，类似 Python 的 for line in f。
	scanner := bufio.NewScanner(f)

	// scanner.Scan() 读取下一行，有内容返回 true，到末尾返回 false。
	// 等价于 Python 的 for line in f:
	for scanner.Scan() {
		// scanner.Text() 获取当前行内容（不含换行符）。
		// strings.TrimSpace 去掉首尾空白，等价于 Python 的 line.strip()。
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释行（# 开头）。
		// strings.HasPrefix 等价于 Python 的 line.startswith("#")。
		if line == "" || strings.HasPrefix(line, "#") {
			continue // 跳到下一次循环，等价于 Python 的 continue
		}

		// strings.SplitN 按 "=" 分割，最多分成 2 部分。
		// 等价于 Python 的 line.split("=", 1)。
		// 返回 []string（字符串切片，类似 Python 的 list[str]）。
		parts := strings.SplitN(line, "=", 2)

		// len() 获取切片长度，等价于 Python 的 len()。
		// 格式不对（没有 = 号）则跳过。
		if len(parts) != 2 {
			continue
		}

		// 同时给两个变量赋值（Go 支持多重赋值，类似 Python 的 a, b = ...）。
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

		// 只有环境变量还没有设置时才写入，避免覆盖系统已有的环境变量。
		// os.Getenv 获取环境变量，不存在时返回空字符串 ""。
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

// fatal 向标准错误输出错误信息，然后以退出码 1 退出程序。
//
// 参数说明：
//   - format string：格式字符串，类似 Python 的 f"..." 或 "".format()
//   - args ...any：可变参数，... 表示可以传任意数量的参数，any 表示任意类型。
//     等价于 Python 的 *args，但 Go 要求声明类型（any 是所有类型的接口）。
func fatal(format string, args ...any) {
	// fmt.Fprintf 向指定的 Writer 写入格式化文本。
	// os.Stderr 是标准错误流，类似 Python 的 sys.stderr。
	// 格式化占位符：%s 字符串，%d 整数，%v 任意类型（Go 自动选择合适格式）。
	fmt.Fprintf(os.Stderr, "错误："+format+"\n", args...)

	// os.Exit(1) 立即退出程序，退出码 1 表示出错。
	// 等价于 Python 的 sys.exit(1)。
	os.Exit(1)
}

// main 是程序入口，Go 程序必须有且只有一个 main 函数。
// 等价于 Python 的 if __name__ == "__main__": 代码块。
func main() {
	// 尝试从两个位置加载 .env 文件：
	// 1. 当前工作目录（在项目目录运行时有效）
	// 2. 固定的项目路径（从任意目录运行时有效）
	loadDotEnv(".env")

	// os.Args 是命令行参数切片，类似 Python 的 sys.argv。
	// os.Args[0] 是程序本身的路径，os.Args[1:] 是用户传入的参数。
	// [1:] 是切片操作，等价于 Python 的 sys.argv[1:]。
	args := os.Args[1:]

	// 没有参数时打印帮助信息并退出。
	if len(args) == 0 {
		// fmt.Println 输出字符串并自动换行，等价于 Python 的 print()。
		fmt.Println(helpText)
		os.Exit(0) // 退出码 0 表示正常退出
	}

	// os.Getenv 读取环境变量，返回字符串。不存在则返回 ""。
	ip := os.Getenv("LAMP_IP")
	token := os.Getenv("LAMP_TOKEN")

	// 校验必填配置是否存在。
	if ip == "" || token == "" {
		// []string{} 创建一个空的字符串切片，类似 Python 的 []。
		missing := []string{}

		if ip == "" {
			// append 向切片追加元素，等价于 Python 的 list.append()。
			// 注意：Go 的 append 返回新切片，必须重新赋值给原变量。
			// Python 的 append 是原地修改，Go 不是。
			missing = append(missing, "LAMP_IP")
		}
		if token == "" {
			missing = append(missing, "LAMP_TOKEN")
		}

		// strings.Join 等价于 Python 的 ", ".join(missing)。
		fatal(".env 文件中缺少配置项：%s", strings.Join(missing, ", "))
	}

	// NewLamp 是构造函数（Go 没有 class，用函数返回结构体指针代替）。
	// 返回两个值：*Lamp 指针和 error。
	// * 表示指针类型（存储的是变量的内存地址，而不是值本身）。
	// Python 里所有对象都是引用，Go 里需要显式区分值和指针。
	lamp, err := NewLamp(ip, token)
	if err != nil {
		// %v 是"默认格式"，对 error 类型会输出错误信息字符串。
		fatal("%v", err)
	}

	// args[0] 是第一个命令参数（子命令名称）。
	cmd := args[0]

	// switch 语句，类似 Python 的 match/case（Python 3.10+）或 if/elif 链。
	// Go 的 switch 默认不会"穿透"到下一个 case（Python 没有 switch）。
	// 不需要 break，每个 case 执行完自动退出。
	switch cmd {
	case "serve":
		addr := os.Getenv("LAMP_WEB_ADDR")
		if addr == "" {
			addr = ":8080"
		}
		if len(args) > 1 {
			addr = args[1]
		}
		if err := runWebServer(lamp, addr); err != nil {
			fatal("Web 控制台启动失败：%v", err)
		}

	case "开":
		// lamp.TurnOn() 调用 Lamp 结构体的 TurnOn 方法。
		// 等价于 Python 的 lamp.turn_on()。
		// := 在这里重新声明了 err，覆盖了外层的 err（Go 允许在内层作用域重新声明）。
		if err := lamp.TurnOn(); err != nil {
			fatal("台灯控制失败：%v", err)
		}
		fmt.Println("台灯已开")

	case "关":
		if err := lamp.TurnOff(); err != nil {
			fatal("台灯控制失败：%v", err)
		}
		fmt.Println("台灯已关")

	case "状态":
		// GetStatus 返回 *Status 指针和 error。
		// s 是一个指向 Status 结构体的指针。
		s, err := lamp.GetStatus()
		if err != nil {
			fatal("台灯控制失败：%v", err)
		}

		// Go 没有三元表达式（Python 有 x if cond else y），
		// 只能用 if/else 赋值。
		onStr := "关"
		if s.On {
			onStr = "开"
		}

		// s.On、s.Brightness、s.ColorTemp 访问结构体字段。
		// s 是指针，Go 会自动解引用，不需要写 (*s).On。
		// %d 格式化整数，%% 输出字面量 % 符号（不是格式占位符）。
		fmt.Printf("状态：%s\n亮度：%d%%\n色温：%dK\n", onStr, s.Brightness, s.ColorTemp)

	case "亮度":
		// 检查是否提供了亮度值参数。
		if len(args) < 2 {
			fatal("请指定亮度值，例如：lamp 亮度 80")
		}

		// strconv.Atoi 将字符串转为整数，等价于 Python 的 int()。
		// 转换失败时 err != nil（例如传入了 "abc"）。
		v, err := strconv.Atoi(args[1])
		if err != nil {
			fatal("亮度必须是整数，收到：%s", args[1])
		}

		if err := lamp.SetBrightness(v); err != nil {
			fatal("台灯控制失败：%v", err)
		}
		fmt.Printf("亮度已设置：%d%%\n", v)

	case "色温":
		if len(args) < 2 {
			fatal("请指定色温值，例如：lamp 色温 4000")
		}
		k, err := strconv.Atoi(args[1])
		if err != nil {
			fatal("色温必须是整数，收到：%s", args[1])
		}
		if err := lamp.SetColorTemp(k); err != nil {
			fatal("台灯控制失败：%v", err)
		}
		fmt.Printf("色温已设置：%dK\n", k)

	default:
		// default 等价于最后的 else，匹配所有其他情况（场景名称）。

		// 检查命令是否是合法的场景名。
		// scenes 是在 device.go 里定义的 map（字典）。
		// map 查找返回两个值：value 和 ok（bool）。
		// _ 是"空白标识符"，表示忽略这个值（等价于 Python 的 _）。
		// ok 为 false 表示 key 不存在。
		if _, ok := scenes[cmd]; !ok {
			fmt.Fprintf(os.Stderr, "错误：未知命令 '%s'\n\n%s\n", cmd, helpText)
			os.Exit(1)
		}

		// *int 是"指向整数的指针"。
		// var brightnessPtr *int 声明一个指针，初始值是 nil（表示"没有值"）。
		// Go 用指针来表达"可选值"的概念，类似 Python 的 Optional[int] 或 None。
		var brightnessPtr *int

		if len(args) > 1 {
			v, err := strconv.Atoi(args[1])
			if err != nil {
				fatal("亮度必须是整数，收到：%s", args[1])
			}
			// & 取变量 v 的地址，得到 *int 指针。
			// 相当于"把 v 的值包装成一个可选值传出去"。
			brightnessPtr = &v
		}

		if err := lamp.SetScene(cmd, brightnessPtr); err != nil {
			fatal("台灯控制失败：%v", err)
		}

		// fmt.Sprintf 格式化字符串但不输出，返回字符串，等价于 Python 的 f"场景：{cmd}"。
		msg := fmt.Sprintf("场景：%s", cmd)

		// brightnessPtr != nil 检查是否有亮度值（等价于 Python 的 if brightness is not None）。
		// *brightnessPtr 解引用指针，取出实际的整数值（等价于 Python 直接访问变量）。
		if brightnessPtr != nil {
			msg += fmt.Sprintf("，亮度：%d%%", *brightnessPtr)
		}
		fmt.Println(msg)
	}
}
