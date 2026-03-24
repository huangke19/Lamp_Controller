// miio.go 实现小米设备的 MiIO 通信协议。
//
// 协议概述：
// 1. 先发送 Hello 包（握手），设备返回它的 ID 和时间戳
// 2. 用设备 token 对 JSON 指令进行 AES 加密
// 3. 拼装二进制数据包（32字节 header + 16字节 checksum + 加密 payload）
// 4. 通过 UDP 发送到设备的 54321 端口
// 5. 解密设备的响应，解析 JSON 结果
package main

// import 导入所需的标准库包。
// Go 编译器会报错（不是警告）如果 import 了但没有使用的包，
// 这强制保持 import 列表整洁，没有 Python 里常见的"无用导入"问题。
import (
	"bytes"           // 字节操作，例如 bytes.Repeat、bytes.TrimRight
	"crypto/aes"      // AES 对称加密算法
	"crypto/cipher"   // 加密模式（CBC 等），需要和 crypto/aes 配合使用
	"crypto/md5"      // MD5 哈希算法（用于生成加密 key/iv 和校验和）
	"encoding/binary" // 二进制编解码，用于读写大端/小端整数
	"encoding/hex"    // 十六进制字符串与字节互转（token 字符串 → 字节）
	"encoding/json"   // JSON 序列化/反序列化，等价于 Python 的 json 模块
	"fmt"             // 格式化输出和错误创建
	"net"             // 网络连接，用于 UDP 通信
	"os"              // 环境变量读取（调试开关）
	"strings"         // 字符串处理
	"sync"            // 互斥锁，保护并发访问
	"time"            // 时间和超时
)

// helloPacket 是 MiIO 协议的握手包，固定 32 字节。
// 设备收到这个包后会返回它的设备 ID 和当前时间戳。
//
// var 声明包级别变量（在所有函数外部）。
// []byte 是字节切片，类似 Python 的 bytes，但 Go 的切片是可变的。
// {0x21, 0x31, ...} 是切片字面量，用花括号初始化，元素用逗号分隔。
// 0x21 是十六进制字面量（等价于十进制 33），在 Python 里写法相同。
const miioTimeout = 5 * time.Second

var helloPacket = []byte{
	0x21, 0x31, 0x00, 0x20, // 魔数 0x2131 + 包长度 0x0020（32字节）
	0xff, 0xff, 0xff, 0xff, // unknown 字段，全 FF
	0xff, 0xff, 0xff, 0xff, // device_id 字段，全 FF 表示"我不知道你是谁"
	0xff, 0xff, 0xff, 0xff, // 时间戳字段，全 FF
	0xff, 0xff, 0xff, 0xff, // checksum 字段（16字节），全 FF
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
}

func debugEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("LAMP_DEBUG")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func debugf(format string, args ...any) {
	if debugEnabled() {
		fmt.Fprintf(os.Stderr, "[miio-debug] "+format+"\n", args...)
	}
}

// MiIO 是一个结构体（struct），用于封装与小米设备通信所需的状态。
//
// Go 没有 class，用 struct 代替。
// struct 只包含数据字段，方法（函数）单独定义，通过"接收者"关联到结构体。
//
// 字段可见性规则：
//   - 首字母大写（如 IP、Token）= 公开，可以从其他包访问（类似 Python 的 public）
//   - 首字母小写（如 deviceID、stamp）= 私有，只能在同一包内访问（类似 Python 的 _前缀约定）
const handshakeTTL = 5 * time.Minute // 握手缓存有效期，超时后强制重新握手同步设备时间戳

type MiIO struct {
	IP            string     // 设备 IP 地址
	Token         []byte     // 设备 token（已从十六进制字符串解码为字节）
	deviceID      []byte     // 握手后从设备获取的 ID（4字节）
	stamp         uint32     // 握手后从设备获取的时间戳（无符号 32 位整数）
	msgID         int        // 消息序号，每次发送后递增，设备用它匹配请求和响应
	mu            sync.Mutex // 保护并发访问，Send 全程持锁
	handshakeDone bool       // 握手结果已缓存，避免每次 Send 重新握手
	handshakeAt   time.Time  // 上次握手时间，用于判断缓存是否过期
}

// NewMiIO 是 MiIO 的构造函数。
//
// Go 没有 __init__ 方法，约定用 New前缀函数创建结构体实例。
// 返回类型是 (*MiIO, error)：成功时返回指针，失败时返回 nil 和错误。
// 调用方通过检查 error 来判断是否成功，这是 Go 最核心的错误处理模式。
//
// Python 等价：
//
//	class MiIO:
//	    def __init__(self, ip, token):
//	        ...
func NewMiIO(ip, token string) (*MiIO, error) {
	// hex.DecodeString 把十六进制字符串转为字节切片。
	// 例如："d208a096..." → []byte{0xd2, 0x08, 0xa0, 0x96, ...}
	// 等价于 Python 的 bytes.fromhex(token)。
	tokenBytes, err := hex.DecodeString(token)

	// || 是逻辑或，等价于 Python 的 or。
	// token 必须是 32 个十六进制字符（解码后 16 字节）。
	if err != nil || len(tokenBytes) != 16 {
		// fmt.Errorf 创建一个新的 error 对象，%s/%d 等格式化占位符同 fmt.Printf。
		// 等价于 Python 的 raise ValueError("invalid token: ...")。
		// 但 Go 不抛出异常，而是把错误作为返回值返回。
		return nil, fmt.Errorf("invalid token: must be 32 hex characters")
	}

	// & 取结构体字面量的地址，返回指针 *MiIO。
	// 这是 Go 创建结构体指针的标准方式，等价于 Python 的 self = MiIO()。
	// 花括号内是字段初始化，未写的字段使用零值（string → ""，int → 0，[]byte → nil）。
	return &MiIO{IP: ip, Token: tokenBytes, msgID: 1}, nil
}

// Handshake 向设备发送 Hello 包并接收响应，获取 deviceID 和时间戳。
//
// func (m *MiIO) 是"方法接收者"语法，表示这个函数属于 *MiIO 类型。
// m 相当于 Python 的 self，*MiIO 表示接收者是指针（可以修改结构体的字段）。
// 如果写 (m MiIO)（不带 *），则是值接收者，修改不会影响原始结构体。
func (m *MiIO) Handshake() error {
	// net.DialTimeout 建立 UDP 连接，5秒超时。
	// "udp" 是网络类型，fmt.Sprintf 拼接 "192.168.x.x:54321"。
	// 等价于 Python 的 socket.socket(socket.AF_INET, socket.SOCK_DGRAM)。
	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:54321", m.IP), miioTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(miioTimeout))

	// Write 发送数据，_ 忽略返回的"写入字节数"（我们不需要它）。
	if _, err := conn.Write(helloPacket); err != nil {
		return err
	}

	// make([]byte, 1024) 创建一个长度为 1024 的字节切片，用于接收数据。
	// 等价于 Python 的 buf = bytearray(1024)。
	buf := make([]byte, 1024)

	// Read 接收数据，n 是实际读取的字节数，数据存在 buf[:n] 中。
	// 等价于 Python 的 data, addr = sock.recvfrom(1024)。
	n, err := conn.Read(buf)
	if err != nil {
		// %w 是特殊的错误包装占位符，把原始错误包进新错误里，
		// 方便调用方用 errors.Is/As 解包检查（类似 Python 的异常链 raise X from Y）。
		return fmt.Errorf("no response from device (check IP and network): %w", err)
	}
	if n < 32 {
		return fmt.Errorf("hello response too short")
	}

	// 从响应包中提取 deviceID 和时间戳。
	// MiIO 响应包格式（32字节 header）：
	//   [0:2]   魔数 0x2131
	//   [2:4]   包长度
	//   [4:8]   unknown
	//   [8:12]  device_id  ← 我们要的
	//   [12:16] timestamp  ← 我们要的
	//   [16:32] checksum
	//
	// buf[8:12] 是切片操作，取索引 8（含）到 12（不含）的子切片，
	// 等价于 Python 的 buf[8:12]，语法完全一样。
	m.deviceID = buf[8:12]

	// binary.BigEndian.Uint32 把 4 个字节解析为大端序 uint32 整数。
	// 等价于 Python 的 int.from_bytes(buf[12:16], 'big')。
	m.stamp = binary.BigEndian.Uint32(buf[12:16])
	m.handshakeAt = time.Now()
	debugf("handshake ok ip=%s device_id=%s stamp=%d", m.IP, hex.EncodeToString(m.deviceID), m.stamp)
	return nil // 返回 nil 表示没有错误，等价于 Python 的 return（没有异常）
}

// Send 发送 JSON-RPC 命令到设备并返回响应结果。
//
// method：RPC 方法名，例如 "set_properties" 或 "get_properties"
// params：方法参数，any 类型表示可以是任意类型（[]prop 切片等）
// 返回 json.RawMessage（原始 JSON 字节）和 error
//
// json.RawMessage 是 []byte 的别名，表示"还没有解析的原始 JSON 数据"，
// 让调用方自己决定如何解析，提供了灵活性。
const maxRetries = 3 // 最多尝试次数（首次 + 2次重试）

func (m *MiIO) Send(method string, params any) (json.RawMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// 首次调用、上次出错后、握手缓存超过 TTL，或重试时，重新握手同步设备时间戳。
		if !m.handshakeDone || time.Since(m.handshakeAt) > handshakeTTL {
			if err := m.Handshake(); err != nil {
				lastErr = err
				m.handshakeDone = false
				debugf("handshake attempt %d/%d failed: %v", attempt+1, maxRetries, err)
				continue // 握手也参与重试，第一个 hello 可能只是唤醒设备 WiFi
			}
			m.handshakeDone = true
		}

		result, err := m.sendOnce(method, params)
		if err == nil {
			return result, nil
		}
		lastErr = err
		debugf("send attempt %d/%d failed: %v", attempt+1, maxRetries, err)
		m.handshakeDone = false // 强制下次重试重新握手
	}
	return nil, lastErr
}

// sendOnce 执行一次实际的 UDP 发包和接收，不含重试逻辑。
// 调用方（Send）负责重试和握手管理。
func (m *MiIO) sendOnce(method string, params any) (json.RawMessage, error) {
	m.msgID++

	payload, err := json.Marshal(map[string]any{
		"id":     m.msgID,
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}
	debugf("request id=%d method=%s params=%s", m.msgID, method, mustJSON(params))

	payload = append(payload, 0x00)

	encrypted, err := miioEncrypt(payload, m.Token)
	if err != nil {
		return nil, err
	}

	pktLen := uint16(32 + len(encrypted))

	// 基于握手时间和实际经过的秒数计算 stamp，保持与设备时钟同步。
	// 设备的 stamp 是秒级递增的计数器，每秒 +1。
	// 之前的 m.stamp++ 每次命令只 +1，但命令间隔可能有 15 秒，
	// 导致 stamp 快速漂移，设备静默丢弃偏差过大的包 → i/o timeout。
	elapsed := uint32(time.Since(m.handshakeAt).Seconds()) + 1
	stamp := m.stamp + elapsed

	header := make([]byte, 16)
	header[0], header[1] = 0x21, 0x31
	binary.BigEndian.PutUint16(header[2:4], pktLen)
	binary.BigEndian.PutUint32(header[4:8], 0x00000000)
	copy(header[8:12], m.deviceID)
	binary.BigEndian.PutUint32(header[12:16], stamp)

	h := md5.New()
	h.Write(header)
	h.Write(m.Token)
	h.Write(encrypted)
	checksum := h.Sum(nil)

	packet := append(header, checksum...)
	packet = append(packet, encrypted...)

	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:54321", m.IP), miioTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(miioTimeout))

	if _, err := conn.Write(packet); err != nil {
		return nil, err
	}
	debugf("packet sent len=%d encrypted_len=%d", len(packet), len(encrypted))

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("no response from device: %w", err)
	}
	if n < 32 {
		return nil, fmt.Errorf("response too short")
	}

	decrypted, err := miioDecrypt(buf[32:n], m.Token)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	decrypted = bytes.TrimRight(decrypted, "\x00")
	debugf("response raw=%s", string(decrypted))

	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(decrypted, &resp); err != nil {
		return nil, fmt.Errorf("invalid response JSON: %w", err)
	}

	if resp.Error != nil {
		m.handshakeDone = false
		return nil, fmt.Errorf("device error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	debugf("response result=%s", string(resp.Result))

	return resp.Result, nil
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("<marshal error: %v>", err)
	}
	return string(b)
}

// miioKeyIV 根据 token 计算 AES 加密所需的 key 和 iv。
//
// 算法：
//
//	key = MD5(token)
//	iv  = MD5(key + token)
//
// 这个函数返回两个值：key 和 iv（都是 []byte）。
// Go 支持多返回值，不需要像 Python 一样 return (key, iv) 再解包。
func miioKeyIV(token []byte) (key, iv []byte) {
	// md5.Sum 计算 MD5，返回 [16]byte（固定长度数组，不是切片）。
	// [16]byte 和 []byte 不同：前者是固定大小的值类型，后者是引用类型。
	k := md5.Sum(token)

	// k[:] 把数组转为切片，等价于 Python 的 bytes(k)。
	// 这是 Go 中数组转切片的标准方式。
	key = k[:]

	// append(key, token...) 拼接两个切片，... 展开 token 切片。
	// 等价于 Python 的 key + token。
	i := md5.Sum(append(key, token...))
	iv = i[:]
	return // 命名返回值，直接 return 不需要写变量名（Go 特有语法）
}

// miioEncrypt 用 AES-128-CBC 加密数据。
func miioEncrypt(plaintext, token []byte) ([]byte, error) {
	key, iv := miioKeyIV(token)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES cipher init failed: %w", err)
	}

	padded := pkcs7Pad(plaintext, aes.BlockSize)
	dst := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(dst, padded)
	return dst, nil
}

// miioDecrypt 用 AES-128-CBC 解密数据。
func miioDecrypt(ciphertext, token []byte) ([]byte, error) {
	key, iv := miioKeyIV(token)

	// AES-CBC 要求密文长度是块大小的整数倍，否则数据损坏。
	// % 是取余运算符，等价于 Python 的 %。
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length not a multiple of block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES cipher init failed: %w", err)
	}
	dst := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(dst, ciphertext)

	// 解密后去掉 PKCS7 填充，返回原始数据。
	return pkcs7Unpad(dst)
}

// pkcs7Pad 对数据进行 PKCS7 填充。
//
// PKCS7 规则：填充 N 个值为 N 的字节，使总长度是 blockSize 的整数倍。
// 例如：数据长度 13，blockSize 16，需要填充 3 个 0x03。
func pkcs7Pad(data []byte, blockSize int) []byte {
	// 计算需要填充的字节数（1 到 blockSize 之间）。
	pad := blockSize - len(data)%blockSize

	// bytes.Repeat([]byte{byte(pad)}, pad) 创建填充字节切片。
	// byte(pad) 把 int 转为 byte 类型（Go 需要显式类型转换）。
	// 等价于 Python 的 bytes([pad] * pad)。
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

// pkcs7Unpad 移除 PKCS7 填充，返回原始数据。
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	// 最后一个字节的值就是填充长度。
	// data[len(data)-1] 取最后一个元素，等价于 Python 的 data[-1]。
	// int() 类型转换，byte → int。
	pad := int(data[len(data)-1])

	// 检查填充值是否合法（1 到 16 之间，且不超过数据长度）。
	if pad == 0 || pad > aes.BlockSize || pad > len(data) {
		return nil, fmt.Errorf("invalid padding")
	}

	// 返回去掉填充部分的切片。
	// data[:len(data)-pad] 等价于 Python 的 data[:-pad]。
	return data[:len(data)-pad], nil
}
