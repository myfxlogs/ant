//go:build windows

package main

import (
	"fmt"
	"unsafe"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/address"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/hdwallet"

	"golang.org/x/sys/windows"
)

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
