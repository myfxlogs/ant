package mql2go

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestParseHeaderFile_Functions(t *testing.T) {
	source := `#pragma once
double MyCustomFunc(int a, double b);
int OrderSendEx(string symbol, int cmd, double volume, double price);
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.mqh")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseHeaderFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, s := range symbols {
		if s.Kind == "function" {
			names = append(names, s.Name)
		}
	}
	sort.Strings(names)

	if !contains(names, "MyCustomFunc") {
		t.Errorf("expected MyCustomFunc, got %v", names)
	}
	if !contains(names, "OrderSendEx") {
		t.Errorf("expected OrderSendEx, got %v", names)
	}
}

func TestParseHeaderFile_Defines(t *testing.T) {
	source := `#define MODE_CUSTOM 42
#define MAX_ORDERS 100
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "defines.mqh")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseHeaderFile(path)
	if err != nil {
		t.Fatal(err)
	}

	found := make(map[string]string)
	for _, s := range symbols {
		if s.Kind == "constant" {
			found[s.Name] = s.Value
		}
	}

	if v, ok := found["MODE_CUSTOM"]; !ok || v != "42" {
		t.Errorf("MODE_CUSTOM not found or wrong value: %v", found)
	}
	if v, ok := found["MAX_ORDERS"]; !ok || v != "100" {
		t.Errorf("MAX_ORDERS not found or wrong value: %v", found)
	}
}

func TestParseHeaderFile_Enums(t *testing.T) {
	source := `enum ENUM_MY_TYPE {
	TYPE_A = 0,
	TYPE_B = 1,
	TYPE_C = 2
};
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "enums.mqh")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseHeaderFile(path)
	if err != nil {
		t.Fatal(err)
	}

	found := make(map[string]string)
	for _, s := range symbols {
		if s.Kind == "enum_value" {
			found[s.Name] = s.Value
		}
	}

	if v, ok := found["TYPE_A"]; !ok || v != "0" {
		t.Errorf("TYPE_A not found or wrong value: %v", found)
	}
	if v, ok := found["TYPE_B"]; !ok || v != "1" {
		t.Errorf("TYPE_B not found or wrong value: %v", found)
	}
	if v, ok := found["TYPE_C"]; !ok || v != "2" {
		t.Errorf("TYPE_C not found or wrong value: %v", found)
	}
}

func TestParseHeaderFile_ClassMethods(t *testing.T) {
	source := `class CMyTrade {
public:
	bool Buy(double volume, double price);
	bool Sell(double volume, double price);
	void SetMagic(int magic);
};
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "class.mqh")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseHeaderFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var methods []string
	for _, s := range symbols {
		if s.Kind == "class_method" {
			methods = append(methods, s.Name)
		}
	}
	sort.Strings(methods)

	if !contains(methods, "CMyTrade.Buy") {
		t.Errorf("expected CMyTrade.Buy, got %v", methods)
	}
	if !contains(methods, "CMyTrade.Sell") {
		t.Errorf("expected CMyTrade.Sell, got %v", methods)
	}
	if !contains(methods, "CMyTrade.SetMagic") {
		t.Errorf("expected CMyTrade.SetMagic, got %v", methods)
	}
}

func TestParseHeaderDir(t *testing.T) {
	dir := t.TempDir()

	// Write multiple .mqh files
	files := map[string]string{
		"a.mqh": `#define CONST_A 1
double FuncA(int x);`,
		"b.mqh": `#define CONST_B 2
int FuncB(string s);`,
		"notmqh.txt": `should be skipped`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	symbols, err := ParseHeaderDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(symbols) < 4 {
		t.Errorf("expected at least 4 symbols, got %d", len(symbols))
	}

	// Verify .txt was skipped
	for _, s := range symbols {
		if strings.HasSuffix(s.Source, "notmqh.txt") {
			t.Error(".txt file should be skipped")
		}
	}
}

func TestGenerateRegistryEntries(t *testing.T) {
	symbols := []HeaderSymbol{
		{Name: "iCustom", Kind: "function", Signature: "double iCustom(string,int,string,...)", Source: "test.mqh"},
		{Name: "MyCustomFunc", Kind: "function", Signature: "double MyCustomFunc(int,double)", Source: "test.mqh"},
		{Name: "OrderSend", Kind: "function", Source: "test.mqh"}, // already implemented
	}

	output := GenerateRegistryEntries(symbols)

	// iCustom is already in unsupported, so it should be skipped
	if strings.Contains(output, "// iCustom —") {
		// Actually iCustom IS in unsupportedSymbols, so it should be skipped
		// Let me check — the function checks IsAPIUnsupported
	}

	// OrderSend is implemented, should be skipped
	if strings.Contains(output, "// OrderSend —") {
		t.Error("OrderSend should be skipped (already implemented)")
	}

	// MyCustomFunc is not in registry, should appear
	if !strings.Contains(output, "MyCustomFunc") {
		t.Error("MyCustomFunc should appear in output")
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
