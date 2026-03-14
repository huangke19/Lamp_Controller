package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

var helloPacket = []byte{
	0x21, 0x31, 0x00, 0x20,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
}

type MiIO struct {
	IP       string
	Token    []byte
	deviceID []byte
	stamp    uint32
	msgID    int
}

func NewMiIO(ip, token string) (*MiIO, error) {
	tokenBytes, err := hex.DecodeString(token)
	if err != nil || len(tokenBytes) != 16 {
		return nil, fmt.Errorf("invalid token: must be 32 hex characters")
	}
	return &MiIO{IP: ip, Token: tokenBytes, msgID: 1}, nil
}

func (m *MiIO) Handshake() error {
	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:54321", m.IP), 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write(helloPacket); err != nil {
		return err
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("no response from device (check IP and network): %w", err)
	}
	if n < 32 {
		return fmt.Errorf("hello response too short")
	}

	m.deviceID = buf[8:12]
	m.stamp = binary.BigEndian.Uint32(buf[12:16])
	return nil
}

func (m *MiIO) Send(method string, params any) (json.RawMessage, error) {
	if err := m.Handshake(); err != nil {
		return nil, err
	}

	m.msgID++
	payload, err := json.Marshal(map[string]any{
		"id":     m.msgID,
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}

	// Append null byte as python-miio does
	payload = append(payload, 0x00)
	encrypted := miioEncrypt(payload, m.Token)

	pktLen := uint16(32 + len(encrypted))
	stamp := m.stamp + 1

	// Build header (first 16 bytes without checksum)
	header := make([]byte, 16)
	header[0], header[1] = 0x21, 0x31
	binary.BigEndian.PutUint16(header[2:4], pktLen)
	binary.BigEndian.PutUint32(header[4:8], 0x00000000)
	copy(header[8:12], m.deviceID)
	binary.BigEndian.PutUint32(header[12:16], stamp)

	// Checksum = MD5(header + token + encrypted_payload)
	h := md5.New()
	h.Write(header)
	h.Write(m.Token)
	h.Write(encrypted)
	checksum := h.Sum(nil)

	packet := append(header, checksum...)
	packet = append(packet, encrypted...)

	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:54321", m.IP), 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write(packet); err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("device error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}

func miioKeyIV(token []byte) (key, iv []byte) {
	k := md5.Sum(token)
	key = k[:]
	i := md5.Sum(append(key, token...))
	iv = i[:]
	return
}

func miioEncrypt(plaintext, token []byte) []byte {
	key, iv := miioKeyIV(token)
	block, _ := aes.NewCipher(key)
	padded := pkcs7Pad(plaintext, aes.BlockSize)
	dst := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(dst, padded)
	return dst
}

func miioDecrypt(ciphertext, token []byte) ([]byte, error) {
	key, iv := miioKeyIV(token)
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length not a multiple of block size")
	}
	block, _ := aes.NewCipher(key)
	dst := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(dst, ciphertext)
	return pkcs7Unpad(dst)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(data) {
		return nil, fmt.Errorf("invalid padding")
	}
	return data[:len(data)-pad], nil
}
