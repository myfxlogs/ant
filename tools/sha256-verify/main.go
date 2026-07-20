package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// selfHashLayout: 32-byte sentinel + 64-byte hash placeholder (patched post-build).
var selfHashLayout = [96]byte{
	'S', 'E', 'L', 'F', 'H', 'S', 'H', 'M', 'A', 'R', 'K', 'E', 'R', '_', '_', '_',
	'X', 'X', 'X', 'X', 'X', 'X', 'X', 'X', 'X', 'X', 'X', 'X', 'X', 'X', 'X', 'X',
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")

	procGetModuleHandle   = kernel32.NewProc("GetModuleHandleW")
	procMessageBoxW       = user32.NewProc("MessageBoxW")
)

const (
	MB_OK              = 0x00000000
	MB_ICONINFORMATION = 0x00000040
	MB_ICONERROR       = 0x00000010
	MB_SETFOREGROUND   = 0x00010000
)

func main() {
	exePath, _ := os.Executable()
	dir := filepath.Dir(exePath)

	var lines []string
	allPass := true

	// --- hdgen.exe ---
	allPass = verify(dir, "hdgen.exe",
		"0b7b4c94f34cecbf898904b692a415166cd963b7dace0b34e42327cd395b27ca",
		&lines, allPass)

	// --- coldsign.exe ---
	allPass = verify(dir, "coldsign.exe",
		"ef61047c8ae9b2a2aac81c16618cbd5724c08f192c20e00609a3d60312aedc5e",
		&lines, allPass)

	// --- verify.exe (self) ---
	allPass = verifySelf(exePath, &lines, allPass)

	// Build message
	body := ""
	if allPass {
		body = "全部文件验证通过，未被篡改，可安全使用。\r\n\r\n"
	} else {
		body = "存在不匹配的文件，请勿使用！\r\n\r\n"
	}
	for _, l := range lines {
		body += l + "\r\n"
	}

	title, _ := windows.UTF16PtrFromString("SHA256 文件完整性验证器")
	msg, _ := windows.UTF16PtrFromString(body)
	flags := uintptr(MB_OK | MB_SETFOREGROUND)
	if allPass {
		flags |= MB_ICONINFORMATION
	} else {
		flags |= MB_ICONERROR
	}
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(msg)), uintptr(unsafe.Pointer(title)), flags)
}

func verify(dir, name, expected string, lines *[]string, allPass bool) bool {
	path := filepath.Join(dir, name)
	f, err := os.Open(path)
	if err != nil {
		*lines = append(*lines, fmt.Sprintf("%s  未找到 (%v)", name, err))
		return false
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	actual := fmt.Sprintf("%x", h.Sum(nil))
	if actual == expected {
		*lines = append(*lines, fmt.Sprintf("%s  通过  %s", name, actual))
		return allPass
	}
	*lines = append(*lines, fmt.Sprintf("%s  不匹配!", name))
	*lines = append(*lines, fmt.Sprintf("  期望: %s", expected))
	*lines = append(*lines, fmt.Sprintf("  实际: %s", actual))
	return false
}

func verifySelf(selfPath string, lines *[]string, allPass bool) bool {
	data, err := os.ReadFile(selfPath)
	if err != nil {
		*lines = append(*lines, fmt.Sprintf("verify.exe  自检失败 (%v)", err))
		return false
	}

	sentinel := selfHashLayout[:32]
	idx := bytes.Index(data, sentinel)
	if idx < 0 {
		*lines = append(*lines, "verify.exe  自检失败 (未找到标记)")
		return false
	}

	hashStart := idx + 32
	if hashStart+64 > len(data) {
		*lines = append(*lines, "verify.exe  自检失败 (标记位置异常)")
		return false
	}

	embedded := string(data[hashStart : hashStart+64])

	zeroed := make([]byte, len(data))
	copy(zeroed, data)
	for i := hashStart; i < hashStart+64; i++ {
		zeroed[i] = 0
	}
	h := sha256.New()
	h.Write(zeroed)
	actual := fmt.Sprintf("%x", h.Sum(nil))

	if actual == embedded {
		*lines = append(*lines, fmt.Sprintf("verify.exe  通过  %s", actual))
		return allPass
	}
	*lines = append(*lines, "verify.exe  不匹配!")
	*lines = append(*lines, fmt.Sprintf("  期望: %s", embedded))
	*lines = append(*lines, fmt.Sprintf("  实际: %s", actual))
	return false
}
