package main

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/go-bip39"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"google.golang.org/protobuf/proto"

	"golang.org/x/sys/windows"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/hdwallet"
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")
	comdlg32 = windows.NewLazySystemDLL("comdlg32.dll")

	procGetModuleHandle    = kernel32.NewProc("GetModuleHandleW")
	procMessageBoxW        = user32.NewProc("MessageBoxW")
	procGetOpenFileNameW   = comdlg32.NewProc("GetOpenFileNameW")
	procGetSaveFileNameW   = comdlg32.NewProc("GetSaveFileNameW")
	procRegisterClassEx    = user32.NewProc("RegisterClassExW")
	procCreateWindowEx     = user32.NewProc("CreateWindowExW")
	procDefWindowProc      = user32.NewProc("DefWindowProcW")
	procGetMessage         = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessage    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procDestroyWindow      = user32.NewProc("DestroyWindow")
	procShowWindow         = user32.NewProc("ShowWindow")
	procUpdateWindow       = user32.NewProc("UpdateWindow")
	procIsDialogMessage    = user32.NewProc("IsDialogMessageW")
	procGetWindowText      = user32.NewProc("GetWindowTextW")
	procGetWindowTextLen   = user32.NewProc("GetWindowTextLengthW")
	procGetDlgItem         = user32.NewProc("GetDlgItem")
)

const (
	CS_HREDRAW          = 2
	CS_VREDRAW          = 1
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_BORDER           = 0x00800000
	WS_TABSTOP          = 0x00010000
	ES_PASSWORD         = 0x0020
	BS_DEFPUSHBUTTON    = 0x0001
	WM_DESTROY          = 0x0002
	WM_CLOSE            = 0x0010
	WM_COMMAND          = 0x0111
	SW_SHOWNORMAL       = 1
	MB_OK               = 0x00000000
	MB_OKCANCEL         = 0x00000001
	MB_ICONWARNING      = 0x00000030
	MB_ICONINFORMATION  = 0x00000040
	MB_ICONERROR        = 0x00000010
	MB_SETFOREGROUND    = 0x00010000
	CW_USEDEFAULT       = 0x80000000
	IDC_ARROW           = 32512
	BN_CLICKED          = 0
	IDCANCEL            = 2
	IDOK                = 1
)

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

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

type POINT struct{ X, Y int32 }
type MSG struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

var (
	mnemonicResult string
	signConfirmed  bool
)

func msgBox(title, body string, flags uintptr) int {
	t, _ := windows.UTF16PtrFromString(title)
	b, _ := windows.UTF16PtrFromString(body)
	ret, _, _ := procMessageBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), flags|MB_SETFOREGROUND)
	return int(ret)
}

func openFileDialog(title string) string {
	filterStr := "Binary files (*.bin)\x00*.bin\x00\x00"
	filter, _ := windows.UTF16PtrFromString(filterStr)
	buf := make([]uint16, 260)
	t, _ := windows.UTF16PtrFromString(title)

	var ofn OPENFILENAME
	ofn.LStructSize = uint32(unsafe.Sizeof(ofn))
	ofn.LpstrFilter = filter
	ofn.LpstrFile = &buf[0]
	ofn.NMaxFile = uint32(len(buf))
	ofn.LpstrTitle = t
	ofn.Flags = 0x00001000 | 0x00000002 | 0x00080000

	ret, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		return ""
	}
	return windows.UTF16ToString(buf)
}

func saveFileDialog(title, defaultName string) string {
	filterStr := "Binary files (*.bin)\x00*.bin\x00All files (*.*)\x00*.*\x00\x00"
	filter, _ := windows.UTF16PtrFromString(filterStr)
	buf := make([]uint16, 260)
	copy(buf, windows.StringToUTF16(defaultName))
	t, _ := windows.UTF16PtrFromString(title)
	ext, _ := windows.UTF16PtrFromString("bin")

	var ofn OPENFILENAME
	ofn.LStructSize = uint32(unsafe.Sizeof(ofn))
	ofn.LpstrFilter = filter
	ofn.LpstrFile = &buf[0]
	ofn.NMaxFile = uint32(len(buf))
	ofn.LpstrTitle = t
	ofn.LpstrDefExt = ext
	ofn.Flags = 0x00000002 | 0x00000800

	ret, _, _ := procGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		return ""
	}
	return windows.UTF16ToString(buf)
}

func main() {
	// Step 1: Select unsigned bundle
	inPath := openFileDialog("选择未签名交易包")
	if inPath == "" {
		return
	}

	// Step 2: Read and parse
	data, err := os.ReadFile(inPath)
	if err != nil {
		msgBox("错误", fmt.Sprintf("读取文件失败: %v", err), MB_OK|MB_ICONERROR)
		return
	}
	var bundle antv1.UnsignedSweepBundle
	if err := proto.Unmarshal(data, &bundle); err != nil {
		msgBox("错误", fmt.Sprintf("解析交易包失败: %v", err), MB_OK|MB_ICONERROR)
		return
	}
	if len(bundle.Txs) == 0 {
		msgBox("错误", "交易包为空。", MB_OK|MB_ICONWARNING)
		return
	}

	// Step 3: Build summary
	coldWallet := ""
	summary := fmt.Sprintf("共 %d 笔待签交易:\r\n\r\n", len(bundle.Txs))
	for i, tx := range bundle.Txs {
		switch tx.GetKind() {
		case antv1.TxKind_TX_KIND_DELEGATE:
			d := tx.GetDelegate()
			summary += fmt.Sprintf("  [%d] Delegate  %s -> %s\r\n",
				i+1, d.GetEnergyAccount(), tx.GetToAddress())
		case antv1.TxKind_TX_KIND_TRANSFER:
			t := tx.GetTransfer()
			amt := tx.GetAmount()
			summary += fmt.Sprintf("  [%d] Transfer  %s -> %s  %s USDT\r\n",
				i+1, tx.GetFromAddress(), tx.GetToAddress(), amt)
			if t.GetAuth() == nil && coldWallet == "" {
				coldWallet = tx.GetToAddress()
			}
		case antv1.TxKind_TX_KIND_UNDELEGATE:
			u := tx.GetUndelegate()
			summary += fmt.Sprintf("  [%d] Undelegate  %s <- %s\r\n",
				i+1, u.GetEnergyAccount(), tx.GetToAddress())
		}
	}

	// Step 4: Confirm
	if msgBox("交易确认", summary+"\r\n确认签名？", MB_OKCANCEL|MB_ICONWARNING) != 1 {
		return
	}

	// Step 5: Mnemonic entry
	mnemonicResult = ""
	signConfirmed = false
	showMnemonicDialog()
	if mnemonicResult == "" || !signConfirmed {
		return
	}
	mnemonicStr := strings.TrimSpace(mnemonicResult)

	if !bip39.IsMnemonicValid(mnemonicStr) {
		msgBox("错误", "助记词无效，请检查后重试。", MB_OK|MB_ICONERROR)
		return
	}

	// Step 6: Sign
	msgs := []string{}
	signed := make([]*antv1.SignedTx, 0, len(bundle.Txs))
	seed := bip39.NewSeed(mnemonicStr, "")

	for i, tx := range bundle.Txs {
		sig, err := deriveAndSignTx(tx, seed, coldWallet)
		if err != nil {
			msgs = append(msgs, fmt.Sprintf("  [%d] 签名失败: %v", i+1, err))
			continue
		}
		signed = append(signed, sig)
		msgs = append(msgs, fmt.Sprintf("  [%d] 已签名  %s -> %s", i+1, sig.GetFromAddress(), sig.GetToAddress()))
	}
	common.ZeroBytes(seed)

	resultSummary := strings.Join(msgs, "\r\n")
	if len(signed) != len(bundle.Txs) {
		msgBox("签名未完全完成", resultSummary+"\r\n\r\n部分交易签名失败，请检查。", MB_OK|MB_ICONERROR)
		return
	}

	// Step 7: Save
	result := &antv1.SignedSweepBundle{Txs: signed}
	outData, err := proto.Marshal(result)
	if err != nil {
		msgBox("错误", fmt.Sprintf("序列化签名包失败: %v", err), MB_OK|MB_ICONERROR)
		return
	}
	outPath := saveFileDialog("保存已签名交易包", "signed.bin")
	if outPath == "" {
		msgBox("提示", "未选择保存路径，签名未导出。", MB_OK|MB_ICONWARNING)
		return
	}
	if err := os.WriteFile(outPath, outData, 0600); err != nil {
		msgBox("错误", fmt.Sprintf("写入文件失败: %v", err), MB_OK|MB_ICONERROR)
		return
	}
	msgBox("签名完成", fmt.Sprintf("已签名 %d 笔交易。\r\n保存至: %s\r\n\r\n%s", len(signed), outPath, resultSummary), MB_OK|MB_ICONINFORMATION)
}

func deriveAndSignTx(tx *antv1.UnsignedTx, seed []byte, coldWalletAddr string) (*antv1.SignedTx, error) {
	var sk *btcec.PrivateKey
	var err error

	switch tx.GetKind() {
	case antv1.TxKind_TX_KIND_TRANSFER:
		sk, err = hdwallet.DeriveDepositPrivKey(seed, tx.GetDerivationIndex())
		if err != nil {
			return nil, fmt.Errorf("derive privkey: %w", err)
		}
		derivedAddr := address.BTCECPrivkeyToAddress(sk).String()
		if derivedAddr != tx.GetFromAddress() {
			return nil, fmt.Errorf("地址不匹配: derived=%s expected=%s", derivedAddr, tx.GetFromAddress())
		}
		// R4 whitelist
		transferTx := tx.GetTransfer()
		if transferTx.GetAuth() == nil && coldWalletAddr != "" && tx.GetToAddress() != coldWalletAddr {
			return nil, fmt.Errorf("R4 白名单: %s != %s", tx.GetToAddress(), coldWalletAddr)
		}

	case antv1.TxKind_TX_KIND_DELEGATE, antv1.TxKind_TX_KIND_UNDELEGATE:
		sk, err = hdwallet.DeriveEnergyAccountPrivKey(seed)
		if err != nil {
			return nil, fmt.Errorf("derive energy privkey: %w", err)
		}
		derivedAddr := address.BTCECPrivkeyToAddress(sk).String()
		if derivedAddr != tx.GetFromAddress() {
			return nil, fmt.Errorf("能量账户地址不匹配: derived=%s expected=%s", derivedAddr, tx.GetFromAddress())
		}

	default:
		return nil, fmt.Errorf("unknown tx kind: %v", tx.GetKind())
	}

	// Sign the raw transaction (ADR-0026 §10.2).
	signedData, txid, err := hdwallet.SignTronTransaction(tx.GetRawTx(), sk)
	if err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	return &antv1.SignedTx{
		Kind:         tx.GetKind(),
		FromAddress:  tx.GetFromAddress(),
		ToAddress:    tx.GetToAddress(),
		Amount:       tx.GetAmount(),
		SignedTxData: signedData,
		TxHash:       txid,
	}, nil
}

// --- Mnemonic dialog ---

func showMnemonicDialog() {
	hInstance := windows.Handle(getModuleHandle())
	className, _ := windows.UTF16PtrFromString("ColdsignMnemonicDlg")
	windowName, _ := windows.UTF16PtrFromString("输入助记词")

	var wc WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.Style = CS_HREDRAW | CS_VREDRAW
	wc.LpfnWndProc = windows.NewCallback(mnemonicWndProc)
	wc.HInstance = hInstance
	wc.HCursor = windows.Handle(loadCursor(0, IDC_ARROW))
	wc.HbrBackground = windows.Handle(getStockObject(5))
	wc.LpszClassName = className
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := procCreateWindowEx.Call(
		0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(windowName)),
		WS_OVERLAPPEDWINDOW&^0x00040000, // no resize
		CW_USEDEFAULT, CW_USEDEFAULT, 520, 180,
		0, 0, uintptr(hInstance), 0,
	)

	createControl("STATIC", "请输入 BIP39 24 词助记词（用空格分隔）：",
		WS_CHILD|WS_VISIBLE, 20, 15, 470, 20, hwnd, 100, hInstance)
	createControl("EDIT", "",
		WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_PASSWORD,
		20, 45, 470, 24, hwnd, 101, hInstance)
	createControl("BUTTON", "确定",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_DEFPUSHBUTTON,
		320, 85, 80, 28, hwnd, IDOK, hInstance)
	createControl("BUTTON", "取消",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP,
		410, 85, 80, 28, hwnd, IDCANCEL, hInstance)

	procShowWindow.Call(hwnd, SW_SHOWNORMAL)
	procUpdateWindow.Call(hwnd)

	var msg MSG
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 || ret == ^uintptr(0) {
			break
		}
		dlgRet, _, _ := procIsDialogMessage.Call(hwnd, uintptr(unsafe.Pointer(&msg)))
		if dlgRet != 0 {
			continue
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func mnemonicWndProc(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		if uint32(wParam>>16) == BN_CLICKED {
			id := uint32(wParam & 0xFFFF)
			switch id {
			case IDOK:
				editHwnd, _, _ := procGetDlgItem.Call(uintptr(hwnd), 101)
				textLen, _, _ := procGetWindowTextLen.Call(editHwnd)
				buf := make([]uint16, textLen+1)
				procGetWindowText.Call(editHwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(textLen+1))
				mnemonicResult = windows.UTF16ToString(buf)
				signConfirmed = true
				procDestroyWindow.Call(uintptr(hwnd))
				return 0
			case IDCANCEL:
				procDestroyWindow.Call(uintptr(hwnd))
				return 0
			}
		}
	case WM_CLOSE:
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func createControl(className, text string, style uint32, x, y, w, h int32, parent uintptr, id int, hInst windows.Handle) uintptr {
	c, _ := windows.UTF16PtrFromString(className)
	t, _ := windows.UTF16PtrFromString(text)
	hwnd, _, _ := procCreateWindowEx.Call(
		0, uintptr(unsafe.Pointer(c)), uintptr(unsafe.Pointer(t)),
		uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, uintptr(id), uintptr(hInst), 0,
	)
	return hwnd
}

func loadCursor(hInst uintptr, id uintptr) uintptr {
	ret, _, _ := user32.NewProc("LoadCursorW").Call(hInst, id)
	return ret
}

func getStockObject(fnObject int32) uintptr {
	ret, _, _ := kernel32.NewProc("GetStockObject").Call(uintptr(fnObject))
	return ret
}

func getModuleHandle() uintptr {
	ret, _, _ := procGetModuleHandle.Call(0)
	return ret
}
