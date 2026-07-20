package main

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/fbsobreira/go-bip39"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/hdwallet"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	comdlg32 = windows.NewLazySystemDLL("comdlg32.dll")

	procMessageBoxW      = user32.NewProc("MessageBoxW")
	procGetSaveFileNameW = comdlg32.NewProc("GetSaveFileNameW")
)

const (
	MB_OK                = 0x00000000
	MB_OKCANCEL          = 0x00000001
	MB_ICONWARNING       = 0x00000030
	MB_ICONINFORMATION   = 0x00000040
	MB_ICONERROR         = 0x00000010
	MB_SETFOREGROUND     = 0x00010000
)

type OPENFILENAME struct {
	LStructSize       uint32
	HwndOwner         windows.Handle
	HInstance         windows.Handle
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData          uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
}

func msgBox(title, body string, flags uintptr) int {
	t, _ := windows.UTF16PtrFromString(title)
	b, _ := windows.UTF16PtrFromString(body)
	ret, _, _ := procMessageBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), flags|MB_SETFOREGROUND)
	return int(ret)
}

func main() {
	// Step 1: Security warning
	if msgBox("HD 钱包初始化",
		"即将生成 24 词助记词和 account xpub。\r\n\r\n"+
			"请确保：\r\n"+
			"  1. 当前在气隙机上运行（无网络连接）\r\n"+
			"  2. 周围无人、无摄像头\r\n"+
			"  3. 准备好金属板或纸张抄写助记词\r\n\r\n"+
			"助记词是恢复全部资金的唯一凭证，\r\n"+
			"请勿拍照、截图或以任何电子形式保存。",
		MB_OKCANCEL|MB_ICONWARNING) != 1 {
		return
	}

	// Step 1.5: One-time-only warning
	if msgBox("⚠ 仅需运行一次",
		"此工具仅需运行一次！\r\n\r\n"+
			"如果之前已生成过助记词并导入 xpub 到在线机，\r\n"+
			"再次运行将生成全新的种子，旧种子对应的地址\r\n"+
			"将无法被监控，充值将检测不到。\r\n\r\n"+
			"是否确认这是首次运行（或确认要替换旧种子）？",
		MB_OKCANCEL|MB_ICONWARNING) != 1 {
		return
	}

	// Step 2: Generate mnemonic
	mnemonic, err := hdwallet.GenerateMnemonic()
	if err != nil {
		msgBox("错误", fmt.Sprintf("生成助记词失败: %v", err), MB_OK|MB_ICONERROR)
		return
	}

	// Step 3: Derive xpub
	seed := bip39.NewSeed(mnemonic, "")
	xpub, fingerprint, err := hdwallet.DeriveAccountXpubAndFingerprint(seed)
	if err != nil {
		msgBox("错误", fmt.Sprintf("派生 xpub 失败: %v", err), MB_OK|MB_ICONERROR)
		return
	}

	// Step 4: Show mnemonic
	msgBox("助记词 — 请立即抄写",
		"以下 24 个词是恢复全部资金的唯一凭证：\r\n\r\n"+
			mnemonic+"\r\n\r\n"+
			"  1. 请用金属板或纸笔抄写，按顺序逐个核对\r\n"+
			"  2. 抄写完毕后关闭此窗口\r\n"+
			"  3. 接下来将保存 xpub.bin 文件",
		MB_OK|MB_ICONINFORMATION)

	// Step 5: Write xpub proto
	export := &antv1.XpubExport{
		Xpub:        xpub,
		Fingerprint: fingerprint,
		Network:     "TRC20",
	}
	data, err := proto.Marshal(export)
	if err != nil {
		msgBox("错误", fmt.Sprintf("序列化 xpub 失败: %v", err), MB_OK|MB_ICONERROR)
		return
	}

	// Step 6: Save file dialog
	outPath := saveFileDialog("xpub.bin")
	if outPath == "" {
		msgBox("提示", "未选择保存路径，xpub 未导出。", MB_OK|MB_ICONWARNING)
		return
	}
	if err := os.WriteFile(outPath, data, 0600); err != nil {
		msgBox("错误", fmt.Sprintf("写入文件失败: %v", err), MB_OK|MB_ICONERROR)
		return
	}

	// Step 7: Done
	msgBox("完成",
		fmt.Sprintf("xpub 已导出到:\r\n%s\r\n\r\nxpub: %s\r\nfingerprint: %s\r\n\r\n"+
			"xpub 用于在线机配置 deposit_xpub 和 deposit_xpub_fingerprint。", outPath, xpub, fingerprint),
		MB_OK|MB_ICONINFORMATION)
}

func saveFileDialog(defaultName string) string {
	// Build filter: "Xpub files (*.bin)\0*.bin\0All files (*.*)\0*.*\0\0"
	filterStr := "Xpub files (*.bin)\x00*.bin\x00All files (*.*)\x00*.*\x00\x00"
	filter, _ := windows.UTF16PtrFromString(filterStr)

	buf := make([]uint16, 260)
	copy(buf, windows.StringToUTF16(defaultName))
	title, _ := windows.UTF16PtrFromString("保存 xpub 文件")

	var ofn OPENFILENAME
	ofn.LStructSize = uint32(unsafe.Sizeof(ofn))
	ofn.LpstrFilter = filter
	ofn.LpstrFile = &buf[0]
	ofn.NMaxFile = uint32(len(buf))
	ofn.LpstrTitle = title
	ofn.LpstrDefExt, _ = windows.UTF16PtrFromString("bin")
	ofn.Flags = 0x00000002 | 0x00000800 // OFN_OVERWRITEPROMPT | OFN_PATHMUSTEXIST

	ret, _, _ := procGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		return ""
	}
	return windows.UTF16ToString(buf)
}
