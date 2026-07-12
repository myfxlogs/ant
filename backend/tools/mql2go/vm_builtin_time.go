package mql2go

import (
	"time"

	"anttrader/tools/mql2go/interp"
)

// MQL4/MQL5 Date/Time functions — complete implementation.

func builtinTimeGMT(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(int32(time.Now().UTC().Unix())), nil
	}
	return interp.IntVal(int32(vm.ctx.ServerTime() / 1000)), nil
}

func builtinTimeGMTOffset(vm *VM, args []interp.Value) (interp.Value, error) {
	_, offset := time.Now().Local().Zone()
	return interp.IntVal(int32(offset)), nil
}

func builtinTimeDaylightSavings(vm *VM, args []interp.Value) (interp.Value, error) {
	_, offset := time.Now().Local().Zone()
	_, stdOffset := time.Now().UTC().Zone()
	return interp.IntVal(int32(offset - stdOffset)), nil
}

func builtinTimeTradeServer(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(vm.ctx.ServerTime() / 1000)), nil
}

func builtinPeriodSeconds(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(60), nil
	}
	tf := vm.ctx.Timeframe()
	if len(args) > 0 {
		tf = periodToTimeframe(argI(args, 0))
	}
	return interp.IntVal(int32(tfDurationSeconds(tf))), nil
}

func tfDurationSeconds(tf string) int {
	switch tf {
	case "M1":
		return 60
	case "M2":
		return 120
	case "M3":
		return 180
	case "M4":
		return 240
	case "M5":
		return 300
	case "M6":
		return 360
	case "M10":
		return 600
	case "M12":
		return 720
	case "M15":
		return 900
	case "M20":
		return 1200
	case "M30":
		return 1800
	case "H1":
		return 3600
	case "H2":
		return 7200
	case "H3":
		return 10800
	case "H4":
		return 14400
	case "H6":
		return 21600
	case "H8":
		return 28800
	case "H12":
		return 43200
	case "D1":
		return 86400
	case "W1":
		return 604800
	case "MN1":
		return 2592000
	default:
		return 3600
	}
}

// periodToTimeframe is an alias for intToTF (same package, unified conversion).
func periodToTimeframe(period int32) string { return intToTF(period) }

// builtinTimeToStruct converts a datetime to an MqlDateTime struct.
// MQL5: struct MqlDateTime { int year; int mon; int day; int hour; int min; int sec; int day_of_week; int day_of_year; }
func builtinTimeToStruct(vm *VM, args []interp.Value) (interp.Value, error) {
	ts := int64(argI(args, 0))
	t := time.Unix(ts, 0).UTC()
	fields := map[string]interp.Value{
		"year":         interp.IntVal(int32(t.Year())),
		"mon":          interp.IntVal(int32(t.Month())),
		"day":          interp.IntVal(int32(t.Day())),
		"hour":         interp.IntVal(int32(t.Hour())),
		"min":          interp.IntVal(int32(t.Minute())),
		"sec":          interp.IntVal(int32(t.Second())),
		"day_of_week":  interp.IntVal(int32(t.Weekday())),
		"day_of_year":  interp.IntVal(int32(t.YearDay())),
	}
	return interp.Value{Kind: interp.ValClass, Class: &interp.ClassInstance{Fields: fields}}, nil
}

// builtinStructToTime converts an MqlDateTime struct back to a datetime (unix seconds).
func builtinStructToTime(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 1 || args[0].Kind != interp.ValClass || args[0].Class == nil {
		return interp.IntVal(0), nil
	}
	f := args[0].Class.Fields
	year := int(f["year"].ToInt())
	mon := int(f["mon"].ToInt())
	day := int(f["day"].ToInt())
	hour := int(f["hour"].ToInt())
	min := int(f["min"].ToInt())
	sec := int(f["sec"].ToInt())
	t := time.Date(year, time.Month(mon), day, hour, min, sec, 0, time.UTC)
	return interp.IntVal(int32(t.Unix())), nil
}
