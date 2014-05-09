package value

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	gotime "time"
	"unicode"

	"github.com/apmckinlay/gsuneido/util/dnum"
	"github.com/apmckinlay/gsuneido/util/verify"
)

/*
SuDate is a Suneido date/time Value

Represents a readable "local" date and time.
Does not take into account time zones or daylight savings.

It is designed to be efficient to pack and unpack
and to convert to human readable formats.
(Calculations are less common.)
*/
type SuDate struct {
	// 21 bits for year, 4 bits for month (1-12), 5 bits for day (1-31)
	date uint32
	// 10 bits for hour, 6 bits for minute, 6 bits for second, 10 bits for ms
	time uint32
}

var NilDate = SuDate{}

var _ Value = NilDate // confirm it implements Value

func DateTime(date uint32, time uint32) SuDate {
	d := SuDate{date, time}
	if !valid(d.Year(), d.Month(), d.Day(),
		d.Hour(), d.Minute(), d.Second(), d.Millisecond()) {
		return NilDate
	}
	return d
}

// NewDate returns a SuDate value, month is 1-12, day is 1-31
func NewDate(yr int, mon int, day int, hr int, min int, sec int, ms int) SuDate {
	if !valid(yr, mon, day, hr, min, sec, ms) {
		return NilDate
	}
	date := uint32(yr<<9) | uint32(mon<<5) | uint32(day)
	time := uint32(hr<<22) | uint32(min<<16) | uint32(sec<<10) | uint32(ms)
	return DateTime(date, time)
}

/* Now returns a SuDate for the current local date & time */
func Now() SuDate {
	return fromTime(gotime.Now())
}

// FromLiteral returns a SuDate from the Suneido literal format
// i.e. yyyymmdd[.hhmm[ss[mmm]]]
func DateFromLiteral(s string) SuDate {
	if s[0] == '#' {
		s = s[1:]
	}
	datelen := strings.IndexRune(s, '.')
	timelen := 0
	if datelen == -1 {
		datelen = len(s)
	} else {
		timelen = len(s) - datelen - 1
	}
	if datelen != 8 ||
		(timelen != 0 && timelen != 4 && timelen != 6 && timelen != 9) {
		return NilDate
	}

	year := nsub(s, 0, 4)
	month := nsub(s, 4, 6)
	day := nsub(s, 6, 8)

	hour := nsub(s, 9, 11)
	minute := nsub(s, 11, 13)
	second := nsub(s, 13, 15)
	millisecond := nsub(s, 15, 18)

	return NewDate(year, month, day, hour, minute, second, millisecond)
}

func nsub(s string, from int, to int) int {
	if to > len(s) {
		return 0
	}
	i, err := strconv.Atoi(s[from:to])
	if err != nil {
		panic("bad date literal")
	}
	return i
}

func fromTime(t gotime.Time) SuDate {
	return NewDate(t.Year(), int(t.Month()), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000000)
}

func (d SuDate) String() string {
	if d.time == 0 {
		return fmt.Sprintf("#%04d%02d%02d", d.Year(), d.Month(), d.Day())
	}
	s := fmt.Sprintf("#%04d%02d%02d.%02d%02d%02d%03d",
		d.Year(), d.Month(), d.Day(),
		d.Hour(), d.Minute(), d.Second(), d.Millisecond())
	if strings.HasSuffix(s, "00000") {
		return s[0:14]
	}
	if strings.HasSuffix(s, "000") {
		return s[0:16]
	}
	return s
}

func (d SuDate) Equals(other interface{}) bool {
	if d2, ok := other.(SuDate); ok {
		return d == d2
	}
	return false
}

func (d SuDate) Hash() uint32 {
	h := uint32(17)
	h = 31*h + d.date
	h = 31*h + d.time
	return h
}

func (d SuDate) hash2() uint32 {
	return d.Hash()
}

func valid(yr int, mon int, day int, hr int, min int, sec int, ms int) bool {
	if !YEAR.valid(yr) || !MONTH.valid(mon) || !DAY.valid(day) ||
		!HOUR.valid(hr) || !MINUTE.valid(min) ||
		!SECOND.valid(sec) || !MILLISECOND.valid(ms) {
		return false
	}
	t := goTime(yr, mon, day, 0, 0, 0, 0)
	return t.Year() == yr && int(t.Month()) == mon && t.Day() == day
}

// Packable

func (d SuDate) PackSize() int {
	return 9
}

func (d SuDate) Pack(buf []byte) []byte {
	buf = append(buf, DATE)
	buf = packUint32(d.date, buf)
	buf = packUint32(d.time, buf)
	return buf
}

func UnpackDate(buf []byte) SuDate {
	date := unpackUint32(buf)
	time := unpackUint32(buf[4:])
	return SuDate{date, time}
}

/* OffsetUTC returns the offset from local to UTC in minutes */
func OffsetUTC() int {
	t := gotime.Now()
	_, offset := t.Zone()
	return -offset / 60
}

// getters

func (d SuDate) Year() int {
	return int(d.date >> 9)
}

func (d SuDate) Month() int {
	return int((d.date >> 5) & 0xf)
}

func (d SuDate) Day() int {
	return int(d.date & 0x1f)
}

func (d SuDate) Hour() int {
	return int(d.time >> 22)
}

func (d SuDate) Minute() int {
	return int((d.time >> 16) & 0x3f)
}

func (d SuDate) Second() int {
	return int((d.time >> 10) & 0x3f)
}

func (d SuDate) Millisecond() int {
	return int(d.time & 0x3ff)
}

func (d SuDate) Plus(yr int, mon int, day int, hr int, min int, sec int, ms int) SuDate {
	yr += d.Year()
	mon += d.Month()
	day += d.Day()
	hr += d.Hour()
	min += d.Minute()
	sec += d.Second()
	ms += d.Millisecond()
	return normalize(yr, mon, day, hr, min, sec, ms)
}

func normalize(yr int, mon int, day int, hr int, min int, sec int, ms int) SuDate {
	t := goTime(yr, mon, day, hr, min, sec, ms)
	return fromGoTime(t)
}

// WeekDay returns the day of the week - Sun is 0, Sat is 6
func (d SuDate) WeekDay() int {
	return int(d.toGoTime().Weekday())
}

// MinusDays returns the difference between two Dates in days
func (d SuDate) MinusDays(other SuDate) int {
	return (int)(d.jday() - other.jday())
}

func (d SuDate) jday() int64 {
	return julianDayNumber(d.Year(), d.Month(), d.Day())
}

// julianDayNumber returns the time's Julian Day Number
// relative to the epoch 12:00 January 1, 4713 BC, Monday.
// NOTE: based on Go time package code
func julianDayNumber(year, month, day int) int64 {
	a := int64(14-month) / 12
	y := int64(year) + 4800 - a
	m := int64(month) + 12*a - 3
	return int64(day) + (153*m+2)/5 + 365*y + y/4 - y/100 + y/400 - 32045
}

// MinusMs returns the difference between two Dates in milliseconds
//
// WARNING: doing this around daylight savings changes may be problematic
func (d SuDate) MinusMs(other SuDate) int64 {
	if d.date == other.date {
		return d.timeAsMs() - other.timeAsMs()
	} else {
		return d.unix() - other.unix()
	}
}

func (d SuDate) timeAsMs() int64 {
	return int64(d.Millisecond()) +
		int64(1000)*int64(d.Second()+60*(d.Minute()+60*d.Hour()))
}

// Time() returns the time in milliseconds since 1 Jan 1970
func (d SuDate) unix() int64 {
	return d.toGoTime().UnixNano() / 1000000
}

func (d SuDate) toGoTime() gotime.Time {
	return goTime(d.Year(), d.Month(), d.Day(),
		d.Hour(), d.Minute(), d.Second(), d.Millisecond())
}

func goTime(yr int, mon int, day int, hr int, min int, sec int, ms int) gotime.Time {
	return gotime.Date(yr, gotime.Month(mon), day, hr, min, sec, ms*1000000, gotime.Local)
}

func fromGoTime(t gotime.Time) SuDate {
	return NewDate(t.Year(), int(t.Month()), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000000)
}

// Format converts the date to a string in the specified format
func (d SuDate) Format(fmt string) string {
	fmtlen := len(fmt)
	dst := new(bytes.Buffer)
	dst.Grow(fmtlen)
	for i := 0; i < fmtlen; i++ {
		n := 1
		if unicode.IsLetter(rune(fmt[i])) {
			for c := fmt[i]; i+1 < fmtlen && fmt[i+1] == c; i++ {
				n++
			}
		}
		switch fmt[i] {
		case 'y':
			yr := d.Year()
			if n >= 4 {
				add(dst, yr/1000)
			}
			if n >= 3 {
				add(dst, (yr%1000)/100)
			}
			if n >= 2 || (yr%100) > 9 {
				add(dst, (yr%100)/10)
			}
			add(dst, yr%10)
		case 'M':
			mon := d.Month()
			if n > 3 {
				dst.WriteString(months[mon-1])
			} else if n == 3 {
				dst.WriteString(months[mon-1][0:3])
			} else {
				if n >= 2 || mon > 9 {
					add(dst, mon/10)
				}
				add(dst, mon%10)
			}
		case 'd':
			if n > 3 {
				dst.WriteString(days[d.WeekDay()])
			} else if n == 3 {
				dst.WriteString(days[d.WeekDay()][0:3])
			} else {
				if n >= 2 || d.Day() > 9 {
					add(dst, d.Day()/10)
				}
				add(dst, d.Day()%10)
			}
		case 'h': // 12 hour
			hr := d.Hour() % 12
			if hr == 0 {
				hr = 12
			}
			if n >= 2 || hr > 9 {
				add(dst, hr/10)
			}
			add(dst, hr%10)
		case 'H': // 24 hour
			if n >= 2 || d.Hour() > 9 {
				add(dst, d.Hour()/10)
			}
			add(dst, d.Hour()%10)
		case 'm':
			if n >= 2 || d.Minute() > 9 {
				add(dst, d.Minute()/10)
			}
			add(dst, d.Minute()%10)
		case 's':
			if n >= 2 || d.Second() > 9 {
				add(dst, d.Second()/10)
			}
			add(dst, d.Second()%10)
		case 'a':
			if d.Hour() < 12 {
				dst.WriteRune('a')
			} else {
				dst.WriteRune('p')
			}
			if n > 1 {
				dst.WriteRune('m')
			}
		case 'A', 't':
			if d.Hour() < 12 {
				dst.WriteRune('A')
			} else {
				dst.WriteRune('P')
			}
			if n > 1 {
				dst.WriteRune('M')
			}
		case '\'':
			for i++; i < fmtlen && (fmt[i] != '\''); i++ {
				dst.WriteByte(fmt[i])
			}
		case '\\':
			i++
			dst.WriteByte(fmt[i])
		default:
			for ; n > 0; n-- {
				dst.WriteByte(fmt[i])
			}
		}
	}
	return dst.String()
}

func add(dst *bytes.Buffer, i int) {
	dst.WriteByte('0' + byte(i))
}

var months = []string{
	"January", "February", "March", "April", "May", "June", "July",
	"August", "September", "October", "November", "December"}
var days = []string{
	"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday",
	"Saturday"}

// Parse converts a human readable date to a SuDate.
//
// Returns NilDate if it fails.
func ParseDate(s string, order string) SuDate {
	NOTSET := 9999
	year := NOTSET
	month := 0
	day := 0
	hour := NOTSET
	minute := NOTSET
	second := NOTSET
	millisecond := 0

	date_patterns := []string{
		"", // set to supplied order
		"md",
		"dm",
		"dmy",
		"mdy",
		"ymd",
	}

	syspat := getSyspat(order, date_patterns)

	// scan
	const MAXTOKENS = 20
	var typ [MAXTOKENS]minmax
	var tokens [MAXTOKENS]int
	ntokens := 0
	got_time := false
	var prev byte
	for si := 0; si < len(s); {
		c := s[si]
		verify.That(ntokens < MAXTOKENS)
		next := nextWord(s, si)
		if next != "" {
			si += len(next)
			i := 0
			for ; i < 12; i++ {
				if strings.HasPrefix(months[i], next) {
					break
				}
			}
			if i < 12 {
				typ[ntokens] = MONTH
				tokens[ntokens] = i + 1
				ntokens++
			} else if next == "Am" || next == "Pm" {
				if next[0] == 'P' {
					if hour < 12 {
						hour += 12
					}
				} else { // (word[0] == 'A')
					if hour == 12 {
						hour = 0
					}
					if hour > 12 {
						return NilDate
					}
				}
			} else {
				// ignore days of week
				for i = 0; i < 7; i++ {
					if strings.HasPrefix(days[i], next) {
						break
					}
				}
				if i >= 7 {
					return NilDate
				}
			}
		} else if next = nextNumber(s, si); next != "" {
			n, _ := strconv.Atoi(next)
			size := len(next)
			si += size
			c = get(s, si)
			if size == 6 || size == 8 {
				dig := digits{next, 0}
				if size == 6 {
					// date with no separators with yy
					tokens[ntokens] = dig.get(2)
					ntokens++
					tokens[ntokens] = dig.get(2)
					ntokens++
					tokens[ntokens] = dig.get(2)
					ntokens++
				} else if size == 8 {
					// date with no separators with yyyy
					for i := 0; i < 3; i++ {
						if syspat[i] == 'y' {
							tokens[ntokens] = dig.get(4)
						} else {
							tokens[ntokens] = dig.get(2)
						}
						ntokens++
					}
				}
				if c == '.' { // time
					si++
					time := nextNumber(s, si)
					tlen := len(time)
					si += tlen
					if tlen == 4 || tlen == 6 || tlen == 9 {
						dig = digits{time, 0}
						hour = dig.get(2)
						minute = dig.get(2)
						if tlen >= 6 {
							second = dig.get(2)
							if tlen == 9 {
								millisecond = dig.get(3)
							}
						}
					}
				}
			} else if prev == ':' || c == ':' || ampm_ahead(s, si) {
				// time
				got_time = true
				if hour == NOTSET {
					hour = n
				} else if minute == NOTSET {
					minute = n
				} else if second == NOTSET {
					second = n
				} else {
					return NilDate
				}
			} else {
				// date
				tokens[ntokens] = n
				if prev == '\'' {
					typ[ntokens] = YEAR
				}
				ntokens++
			}
		} else {
			prev = c // ignore
			si++
		}
	}
	if hour == NOTSET {
		hour = 0
	}
	if minute == NOTSET {
		minute = 0
	}
	if second == NOTSET {
		second = 0
	}

	// search for date match
	pat := 0
	p := ""
	for ; pat < len(date_patterns); pat++ {
		p = date_patterns[pat]
		// try one pattern
		var t int
		for t = 0; t < len(p) && t < ntokens; t++ {
			var part minmax
			if p[t] == 'y' {
				part = YEAR
			} else if p[t] == 'm' {
				part = MONTH
			} else if p[t] == 'd' {
				part = DAY
			} else {
				panic("shouldn't reach here")
			}
			if (typ[t] != UNK && typ[t] != part) ||
				tokens[t] < part.min || tokens[t] > part.max {
				break
			}
		}
		// stop at first match
		verify.That(p != "")
		if t == len(p) && t == ntokens {
			break
		}
	}
	verify.That(p != "")

	now := Now()

	if pat < len(date_patterns) {
		// use match
		for t := 0; t < len(p); t++ {
			if p[t] == 'y' {
				year = tokens[t]
			} else if p[t] == 'm' {
				month = tokens[t]
			} else if p[t] == 'd' {
				day = tokens[t]
			} else {
				panic("shouldn't reach here")
			}
		}
	} else if got_time && ntokens == 0 {
		year = now.Year()
		month = now.Month()
		day = now.Day()
	} else {
		return NilDate // no match
	}

	if year == NOTSET {
		if month >= max(now.Month()-6, 1) &&
			month <= min(now.Month()+5, 12) {
			year = now.Year()
		} else if now.Month() < 6 {
			year = now.Year() - 1
		} else {
			year = now.Year() + 1
		}
	} else if year < 100 {
		thisyr := now.Year()
		year += thisyr - thisyr%100
		if year-thisyr > 20 {
			year -= 100
		}
	}
	if !valid(year, month, day, hour, minute, second, millisecond) {
		return NilDate
	}
	return NewDate(year, month, day, hour, minute, second, millisecond)
}

func min(x int, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}

func max(x int, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}

func nextWord(s string, si int) string {
	dst := []byte{}
	for ; si < len(s) && unicode.IsLetter(rune(s[si])); si++ {
		dst = append(dst, byte(unicode.ToLower(rune(s[si]))))
	}
	if len(dst) == 0 {
		return ""
	}
	dst[0] = byte(unicode.ToUpper(rune(dst[0])))
	return string(dst)
}

func nextNumber(s string, si int) string {
	i := si
	for ; i < len(s) && unicode.IsDigit(rune(s[i])); i++ {
	}
	return s[si:i]
}

func getSyspat(order string, date_patterns []string) []byte {
	syspat := make([]byte, 3)
	i := 0
	oc := byte(0)
	prev := byte(0)
	for oi := 0; oi < len(order) && i < 3; oi++ {
		oc = order[oi]
		if oc != prev && (oc == 'y' || oc == 'M' || oc == 'd') {
			syspat[i] = byte(unicode.ToLower(rune(oc)))
			i++
		}
		prev = oc
	}
	if i != 3 {
		panic("invalid date format: '" + order + "'")
	}
	date_patterns[0] = string(syspat)

	// swap month-day patterns if system setting is day first
	for i = 0; i < 3; i++ {
		if syspat[i] == 'm' {
			break
		} else if syspat[i] == 'd' {
			date_patterns[1], date_patterns[2] = date_patterns[2], date_patterns[1]
		}
	}
	return syspat
}

func ampm_ahead(s string, i int) bool {
	s0 := get(s, i)
	if s0 == ' ' {
		i++
		s0 = get(s, i)
	}
	s0 = byte(unicode.ToLower(rune(s0)))
	return (s0 == 'a' || s0 == 'p') &&
		unicode.ToLower(rune(get(s, i+1))) == 'm'
}

func get(s string, i int) byte {
	if i < len(s) {
		return s[i]
	} else {
		return 0
	}
}

type digits struct {
	s string
	i int
}

func (d *digits) get(n int) int {
	d.i += n
	i, _ := strconv.Atoi(d.s[d.i-n : d.i])
	return i
}

type minmax struct {
	min int
	max int
}

func (m minmax) valid(n int) bool {
	return m.min <= n && n <= m.max
}

var (
	YEAR        = minmax{0, 3000}
	MONTH       = minmax{1, 12}
	DAY         = minmax{1, 31}
	HOUR        = minmax{0, 23}
	MINUTE      = minmax{0, 59}
	SECOND      = minmax{0, 59}
	MILLISECOND = minmax{0, 999}
	UNK         = minmax{0, 0}
)

// Value interface

func (d SuDate) Get(key Value) Value {
	panic("date does not support get")
}

func (d SuDate) Put(key Value, val Value) {
	panic("date does not support put")
}

func (d SuDate) ToInt() int32 {
	panic("cannot convert date to integer")
}

func (d SuDate) ToDnum() dnum.Dnum {
	panic("cannot convert date to number")
}

func (d SuDate) ToStr() string {
	panic("cannot convert date to string")
}

func (_ SuDate) TypeName() string {
	return "Date"
}

func (_ SuDate) order() Order {
	return OrdDate
}

func (d SuDate) cmp(other Value) int {
	d2 := other.(SuDate)
	if d.date < d2.date {
		return -1
	} else if d.date > d2.date {
		return +1
	} else if d.time < d2.time {
		return -1
	} else if d.time > d2.time {
		return +1
	} else {
		return 0
	}
}
