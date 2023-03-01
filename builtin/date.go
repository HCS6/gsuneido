// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package builtin

import (
	"strings"
	"time"

	. "github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/dnum"
	"github.com/apmckinlay/gsuneido/util/str"
)

type suDateGlobal struct {
	SuBuiltin
}

func init() {
	ps := params(`(string=false, pattern=false,
		year=nil, month=nil, day=nil,
		hour=nil, minute=nil, second=nil, millisecond=nil)`)
	Global.Builtin("Date", &suDateGlobal{SuBuiltin{Fn: Date,
		BuiltinParams: BuiltinParams{ParamSpec: ps}}})
}

func Date(_ *Thread, args []Value) Value {
	if args[0] != False && hasFields(args) {
		panic("usage: Date() or Date(string [, pattern]) or " +
			"Date(year:, month:, day:, hour:, minute:, second:)")
	}
	if args[0] != False {
		if _, ok := args[0].(SuDate); ok {
			return args[0]
		}
		var d SuDate
		s := AsStr(args[0])
		if args[1] == False {
			if strings.HasPrefix(s, "#") {
				d = DateFromLiteral(s)
			} else {
				d = ParseDate(s, "yMd")
			}
		} else {
			d = ParseDate(s, AsStr(args[1]))
		}
		if d == NilDate {
			return False
		}
		return d
	} else if hasFields(args) {
		return named(args)
	}
	return Now()
}

func hasFields(args []Value) bool {
	for i := 2; i <= 8; i++ {
		if args[i] != nil {
			return true
		}
	}
	return false
}

func named(args []Value) Value {
	now := Now()
	year := now.Year()
	month := now.Month()
	day := now.Day()
	hour := now.Hour()
	minute := now.Minute()
	second := now.Second()
	millisecond := now.Millisecond()
	if args[2] != nil {
		year = ToInt(args[2])
	}
	if args[3] != nil {
		month = ToInt(args[3])
	}
	if args[4] != nil {
		day = ToInt(args[4])
	}
	if args[5] != nil {
		hour = ToInt(args[5])
	}
	if args[6] != nil {
		minute = ToInt(args[6])
	}
	if args[7] != nil {
		second = ToInt(args[7])
	}
	if args[8] != nil {
		millisecond = ToInt(args[8])
	}
	d := NormalizeDate(year, month, day, hour, minute, second, millisecond)
	if d == NilDate {
		return False
	}
	return d
}

func (d *suDateGlobal) Get(th *Thread, key Value) Value {
	m := ToStr(key)
	if fn, ok := dateStaticMethods[m]; ok {
		return fn.(Value)
	}
	if fn, ok := ParamsMethods[m]; ok {
		return NewSuMethod(d, fn.(Value))
	}
	return nil
}

func (d *suDateGlobal) Lookup(th *Thread, method string) Callable {
	if fn, ok := dateStaticMethods[method]; ok {
		return fn
	}
	return d.SuBuiltin.Lookup(th, method) // for Params
}

func (d *suDateGlobal) String() string {
	return "Date /* builtin class */"
}

var msFactor = dnum.FromStr(".001")

var dateStaticMethods = methods()

var _ = method(date_Begin, "()")

func date_Begin(Value) Value {
	return DateBegin
}

var _ = method(date_End, "()")

func date_End(Value) Value {
	return DateEnd
}

var _ = exportMethods(&DateMethods)

var _ = method(date_MinusDays, "(date)")

func date_MinusDays(this Value, val Value) Value {
	t1 := this.(SuDate)
	if t2, ok := val.(SuDate); ok {
		return IntVal(t1.MinusDays(t2))
	}
	panic("date.MinusDays requires date")
}

var _ = method(date_MinusSeconds, "(date)")

func date_MinusSeconds(this Value, val Value) Value {
	t1 := this.(SuDate)
	if t2, ok := val.(SuDate); ok {
		if t1.Year()-t2.Year() >= 50 {
			panic("date.MinusSeconds interval too large")
		}
		ms := t1.MinusMs(t2)
		return SuDnum{Dnum: dnum.Mul(dnum.FromInt(ms), msFactor)}
	}
	panic("date.MinusSeconds requires date")
}

var _ = method(date_FormatEn, "(format)")

func date_FormatEn(this, arg Value) Value {
	return SuStr(this.(SuDate).Format(ToStr(arg)))
}

var _ = method(date_GetLocalGMTBias, "()")

func date_GetLocalGMTBias(this Value) Value { // should be static
	_, offset := time.Now().Zone()
	return IntVal(-offset / 60)
}

var _ = method(date_Plus, "(years=0, months=0, days=0, "+
	"hours=0, minutes=0, seconds=0, milliseconds=0)")

func date_Plus(th *Thread, this Value, args []Value) Value {
	return this.(SuDate).Plus(ToInt(args[0]), ToInt(args[1]),
		ToInt(args[2]), ToInt(args[3]), ToInt(args[4]),
		ToInt(args[5]), ToInt(args[6]))
}

var _ = method(date_WeekDay, "(firstDay='Sun')")

func date_WeekDay(this, arg Value) Value {
	i := dayOfWeek(arg)
	return IntVal(((this.(SuDate).WeekDay() - i) + 7) % 7)
}

var _ = method(date_Year, "()")

func date_Year(this Value) Value {
	return IntVal(this.(SuDate).Year())
}

var _ = method(date_Month, "()")

func date_Month(this Value) Value {
	return IntVal(this.(SuDate).Month())
}

var _ = method(date_Day, "()")

func date_Day(this Value) Value {
	return IntVal(this.(SuDate).Day())
}

var _ = method(date_Hour, "()")

func date_Hour(this Value) Value {
	return IntVal(this.(SuDate).Hour())
}

var _ = method(date_Minute, "()")

func date_Minute(this Value) Value {
	return IntVal(this.(SuDate).Minute())
}

var _ = method(date_Second, "()")

func date_Second(this Value) Value {
	return IntVal(this.(SuDate).Second())
}

var _ = method(date_Millisecond, "()")

func date_Millisecond(this Value) Value {
	return IntVal(this.(SuDate).Millisecond())
}

func dayOfWeek(x Value) int {
	if i, ok := x.IfInt(); ok {
		return i
	}
	s := str.ToLower(AsStr(x))
	days := []string{"sunday", "monday", "tuesday",
		"wednesday", "thursday", "friday", "saturday"}
	for i, d := range days {
		if strings.HasPrefix(d, s) {
			return i
		}
	}
	panic("usage: date.WeekDay(day name or number)")
}

var _ = builtin(UnixTime, "()")

func UnixTime() Value {
	return IntVal(int(time.Now().Unix()))
}
