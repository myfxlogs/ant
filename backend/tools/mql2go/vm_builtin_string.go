package mql2go

import (
	"strings"
	"time"

	"alphaforge/tools/mql2go/interp"
)

// ── String builtins ──────────────────────────────────────────────────

func builtinStringFind(vm *VM, args []interp.Value) (interp.Value, error) {
	str := argS(args, 0)
	substr := argS(args, 1)
	start := 0
	if len(args) > 2 {
		start = int(argI(args, 2))
	}
	if start < 0 || start > len(str) {
		return interp.IntVal(-1), nil
	}
	idx := strings.Index(str[start:], substr)
	if idx < 0 {
		return interp.IntVal(-1), nil
	}
	return interp.IntVal(int32(idx + start)), nil
}

func builtinStringSubstr(vm *VM, args []interp.Value) (interp.Value, error) {
	str := argS(args, 0)
	start := int(argI(args, 1))
	if start < 0 {
		start = 0
	}
	if start > len(str) {
		return interp.StringVal(""), nil
	}
	if len(args) > 2 && args[2].ToInt() >= 0 {
		end := start + int(args[2].ToInt())
		if end > len(str) {
			end = len(str)
		}
		return interp.StringVal(str[start:end]), nil
	}
	return interp.StringVal(str[start:]), nil
}

func builtinStringLen(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(int32(len(argS(args, 0)))), nil
}

func builtinStringReplace(vm *VM, args []interp.Value) (interp.Value, error) {
	str := argS(args, 0)
	old := argS(args, 1)
	newStr := argS(args, 2)
	if old == "" {
		return interp.StringVal(str), nil
	}
	return interp.StringVal(strings.ReplaceAll(str, old, newStr)), nil
}

func builtinStringSplit(vm *VM, args []interp.Value) (interp.Value, error) {
	str := argS(args, 0)
	sep := argS(args, 1)
	parts := strings.Split(str, sep)
	return interp.ArrayVal(stringsToValues(parts)), nil
}

func builtinStringTrimLeft(vm *VM, args []interp.Value) (interp.Value, error) {
	str := argS(args, 0)
	chars := " \t\n\r"
	if len(args) > 1 {
		chars = argS(args, 1)
	}
	return interp.StringVal(strings.TrimLeft(str, chars)), nil
}

func builtinStringTrimRight(vm *VM, args []interp.Value) (interp.Value, error) {
	str := argS(args, 0)
	chars := " \t\n\r"
	if len(args) > 1 {
		chars = argS(args, 1)
	}
	return interp.StringVal(strings.TrimRight(str, chars)), nil
}

func builtinStringConcatenate(vm *VM, args []interp.Value) (interp.Value, error) {
	var sb strings.Builder
	for _, a := range args {
		sb.WriteString(a.ToString())
	}
	return interp.StringVal(sb.String()), nil
}

func stringsToValues(ss []string) []interp.Value {
	result := make([]interp.Value, len(ss))
	for i, s := range ss {
		result[i] = interp.StringVal(s)
	}
	return result
}

// ── Datetime builtins ────────────────────────────────────────────────

func builtinDay(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	t := time.UnixMilli(vm.ctx.ServerTime()).UTC()
	return interp.IntVal(int32(t.Day())), nil
}

func builtinDayOfWeek(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	t := time.UnixMilli(vm.ctx.ServerTime()).UTC()
	return interp.IntVal(int32(t.Weekday())), nil
}

func builtinHour(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	t := time.UnixMilli(vm.ctx.ServerTime()).UTC()
	return interp.IntVal(int32(t.Hour())), nil
}

func builtinMinute(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	t := time.UnixMilli(vm.ctx.ServerTime()).UTC()
	return interp.IntVal(int32(t.Minute())), nil
}

func builtinYear(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	t := time.UnixMilli(vm.ctx.ServerTime()).UTC()
	return interp.IntVal(int32(t.Year())), nil
}

func builtinMonth(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	t := time.UnixMilli(vm.ctx.ServerTime()).UTC()
	return interp.IntVal(int32(t.Month())), nil
}

func builtinStrToTime(vm *VM, args []interp.Value) (interp.Value, error) {
	s := argS(args, 0)
	// MQL4 StrToTime expects "yyyy.mm.dd hh:mi" format
	layouts := []string{
		"2006.01.02 15:04",
		"2006.01.02 15:04:05",
		"2006.01.02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return interp.IntVal(int32(t.Unix())), nil
		}
	}
	return interp.IntVal(0), nil
}

func builtinTimeToStr(vm *VM, args []interp.Value) (interp.Value, error) {
	ts := int64(argI(args, 0))
	mode := int32(0)
	if len(args) > 1 {
		mode = argI(args, 1)
	}
	return interp.StringVal(formatMQLTime(ts, mode)), nil
}

// ── Platform builtins ────────────────────────────────────────────────

func builtinIsTesting(vm *VM, args []interp.Value) (interp.Value, error) {
	// VM-TRADE-CONTEXT-2: IsTesting must reflect actual execution mode.
	// signalMode=true means live/paper trading; signalMode=false means backtest.
	return interp.BoolVal(!vm.signalMode), nil
}

func builtinAccountNumber(vm *VM, args []interp.Value) (interp.Value, error) {
	// VM-TRADE-CONTEXT-2: AccountNumber must come from the authoritative
	// account context, not a hardcoded value. In backtest, the SimBroker
	// provides the account login; in live, the executor provides it.
	// If Login is 0 (unavailable), record a blind spot and return 0.
	if vm.ctx == nil {
		vm.recordBlindSpot("AccountNumber")
		return interp.IntVal(0), nil
	}
	login := vm.ctx.Account().Login
	if login == 0 {
		vm.recordBlindSpot("AccountNumber")
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(login)), nil
}

// ── Array builtins (real implementations) ────────────────────────────
