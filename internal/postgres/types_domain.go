package postgres

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Domain is the set of values a column can take given a list of range comparisons against
// constants, e.g.:
// 'c1 > 42 AND c1 <> 50 AND c1 <= 123'. Constants are passed as they come from the SQL query
// and each comparison narrows the domain.
type Domain interface {
	// Above constrains the domain to values greater than constant or, greater or equal when inclusive is true.
	Above(constant string, inclusive bool)
	// Below constrains the domain to values lower than constant or, lower or equal when inclusive is true.
	Below(constant string, inclusive bool)
	// Exclude removes constant from the domain.
	Exclude(constant string)
	// Gen returns a random value from the domain. Never returns nil.
	Gen() any
}

type domainType interface {
	newDomain(*ColumnMeta) Domain
}

const (
	secondsPerDay = 24 * 60 * 60
	// max number of decimal digits that always fit in an int64
	maxInt64Digits = 18
)

var (
	// bounds of time types, kept within the range every time type and encoder supports
	minTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	maxTime = time.Date(2100, 12, 31, 23, 59, 59, 999999000, time.UTC)

	pow10 = func() [maxInt64Digits + 1]int64 {
		var p [maxInt64Digits + 1]int64
		p[0] = 1
		for i := 1; i < len(p); i++ {
			p[i] = p[i-1] * 10
		}
		return p
	}()
)

// intConv converts the values of a type to and from integers, e.g.: timestamps to microseconds.
type intConv interface {
	// toInt maps constant to the largest integer lower or equal than constant and returns whether the conversion is
	// exact, e.g.: 42.5 -> (42, false). Results out of the int64 range are saturated.
	toInt(constant string) (floor int64, exact bool)
	fromInt(n int64) any
}

// intDomain is the domain of types that can be mapped to integers, represented by the inclusive range [lo, hi]
// minus a set of excluded values.
type intDomain struct {
	conv   intConv
	lo, hi int64
	// upper bound used while hi is not constrained by Below, e.g.: Now() for time types. Ignored when lower than lo,
	// e.g.: 'c1 > <future date>', in which case hi is used
	defaultHi int64
	excluded  []int64
	// Se to true after the excluded list has been adjusted for the domain
	excludedForDomain bool
}

// newIntDomain returns a domain in the range [lo, hi] that generates values up to defaultHi unless constrained
// otherwise.
func newIntDomain(conv intConv, lo, hi, defaultHi int64) *intDomain {
	return &intDomain{conv: conv, lo: lo, hi: hi, defaultHi: defaultHi}
}

func (d *intDomain) Above(constant string, inclusive bool) {
	floor, exact := d.conv.toInt(constant)
	lo := floor
	if !inclusive || !exact {
		lo++
	}
	d.lo = max(d.lo, lo)
	d.excludedForDomain = false
}

func (d *intDomain) Below(constant string, inclusive bool) {
	floor, exact := d.conv.toInt(constant)
	hi := floor
	if !inclusive && exact {
		hi--
	}
	d.hi = min(d.hi, hi)
	d.defaultHi = d.hi
	d.excludedForDomain = false
}

func (d *intDomain) Exclude(constant string) {
	// a constant that is not representable by the type is never equal to any of its values
	if n, exact := d.conv.toInt(constant); exact {
		d.excluded = append(d.excluded, n)
		d.excludedForDomain = false
	}
}

// Gen randomly generates a value within the domain.
func (d *intDomain) Gen() any {
	hi := d.hi
	if d.defaultHi >= d.lo {
		// usually defaults to d.defaultHi assuming hi is the max value of the type used
		hi = min(hi, d.defaultHi)
	}

	if !d.excludedForDomain {
		d.excludedInRange(hi)
		d.excludedForDomain = true
	}

	span := uint64(hi) - uint64(d.lo) - uint64(len(d.excluded))
	var r uint64
	if span == math.MaxUint64 {
		r = rand.Uint64()
	} else {
		r = rand.Uint64N(span + 1)
	}

	n := d.lo + int64(r)
	for _, e := range d.excluded {
		if e > n {
			break
		}
		n++
	}

	return d.conv.fromInt(n)
}

// excludedInRange drops the excluded values out of [lo, hi] and returns the remaining ones sorted and without
// duplicates.
func (d *intDomain) excludedInRange(hi int64) {
	d.excluded = slices.DeleteFunc(d.excluded, func(e int64) bool { return e < d.lo || e > hi })
	slices.Sort(d.excluded)
	d.excluded = slices.Compact(d.excluded)
}

// intNConv is the converter for all different ints with different number of bits.
type intNConv struct {
	bits int
}

func (c intNConv) toInt(constant string) (int64, bool) {
	if n, err := strconv.ParseInt(constant, 10, 64); err == nil {
		return n, true
	}

	// e.g.: 'c1 > 42.5'
	return parseScaled(constant, 0)
}

func (c intNConv) fromInt(n int64) any {
	switch c.bits {
	case 16:
		return int16(n)
	case 32:
		return int32(n)
	default:
		return n
	}
}

// numericConv represents numerics by their unscaled value, e.g.: 123.45 with scale 2 -> 12345.
type numericConv struct {
	scale int
}

func (c numericConv) toInt(constant string) (int64, bool) {
	if n, err := strconv.ParseInt(constant, 10, 64); err == nil {
		p := pow10[c.scale]
		if n <= math.MaxInt64/p && n >= math.MinInt64/p {
			return n * p, true
		}
	}

	return parseScaled(constant, c.scale)
}

// fromInt formats n as a decimal string without trailing zeros, the same format used to generate numeric values.
func (c numericConv) fromInt(n int64) any {
	s := strconv.FormatInt(n, 10)
	if c.scale == 0 {
		return s
	}

	sign := ""
	if n < 0 {
		sign, s = "-", s[1:]
	}
	if len(s) <= c.scale {
		s = strings.Repeat("0", c.scale-len(s)+1) + s
	}
	intPart, fracPart := s[:len(s)-c.scale], strings.TrimRight(s[len(s)-c.scale:], "0")
	if fracPart == "" {
		return sign + intPart
	}

	return sign + intPart + "." + fracPart
}

func parseScaled(constant string, scale int) (int64, bool) {
	r, ok := new(big.Rat).SetString(constant)
	if !ok {
		panic(fmt.Errorf("invalid numeric constant %q", constant))
	}
	if scale > 0 {
		r.Mul(r, new(big.Rat).SetInt64(pow10[scale]))
	}

	// the denominator is always positive so, the euclidean division returns the floor
	var rem big.Int
	floor, _ := new(big.Int).DivMod(r.Num(), r.Denom(), &rem)
	if !floor.IsInt64() {
		panic(fmt.Errorf("%s does not fit in an int64", constant))
	}

	return floor.Int64(), rem.Sign() == 0
}

// dateConv represents dates by the number of days since the Unix epoch.
type dateConv struct{}

func (dateConv) toInt(constant string) (int64, bool) {
	t := parseTime(constant)
	sec := t.Unix()
	days := sec / secondsPerDay
	rem := sec % secondsPerDay
	if rem < 0 {
		days--
		rem += secondsPerDay
	}

	return days, rem == 0 && t.Nanosecond() == 0
}

func (dateConv) fromInt(n int64) any {
	return time.Unix(n*secondsPerDay, 0).UTC()
}

// timestampConv represents timestamps by the number of microseconds since the Unix epoch, the resolution of
// Postgres timestamps.
type timestampConv struct{}

func (timestampConv) toInt(constant string) (int64, bool) {
	t := parseTime(constant)

	return t.UnixMicro(), t.Nanosecond()%1000 == 0
}

func (timestampConv) fromInt(n int64) any {
	return time.UnixMicro(n).UTC()
}

var timeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999Z07",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04",
	time.DateOnly,
}

// parseTime parses date and time constants in the formats commonly used in SQL queries. Constants without time zone
// are assumed to be UTC.
func parseTime(constant string) time.Time {
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, constant); err == nil {
			return t
		}
	}

	panic(fmt.Errorf("unsupported date/time format %q", constant))
}

// floatDomain is the domain of floating point types, represented by the inclusive range [lo, hi] minus a set of
// excluded values. float4 values are kept as float64, which represents them exactly.
type floatDomain struct {
	bits     int
	lo, hi   float64
	excluded []float64
}

func (d *floatDomain) Above(constant string, inclusive bool) {
	v := parseFloat(constant)
	lo := d.nearest(v)
	if lo < v || (lo == v && !inclusive) {
		lo = d.next(lo, math.Inf(1))
	}
	d.lo = max(d.lo, lo)
}

func (d *floatDomain) Below(constant string, inclusive bool) {
	v := parseFloat(constant)
	hi := d.nearest(v)
	if hi > v || (hi == v && !inclusive) {
		hi = d.next(hi, math.Inf(-1))
	}
	d.hi = min(d.hi, hi)
}

func (d *floatDomain) Exclude(constant string) {
	// a constant that is not representable by the type is never equal to any of its values
	if v := parseFloat(constant); d.nearest(v) == v {
		d.excluded = append(d.excluded, v)
	}
}

func (d *floatDomain) Gen() any {
	// interpolate instead of 'lo + r*(hi-lo)' as 'hi-lo' overflows for wide ranges
	r := rand.Float64()
	v := min(max(d.nearest(d.lo*(1-r)+d.hi*r), d.lo), d.hi)

	// on collision move to the next value towards the inside of the domain
	if slices.Contains(d.excluded, v) {
		if v < d.hi {
			v = d.next(v, math.Inf(1))
		} else {
			v = d.next(v, math.Inf(-1))
		}
	}

	if d.bits == 32 {
		return float32(v)
	}

	return v
}

// nearest rounds v to the nearest value representable by the type.
func (d *floatDomain) nearest(v float64) float64 {
	if d.bits == 32 {
		return float64(float32(v))
	}

	return v
}

// next returns the next value representable by the type after v in the direction of to.
func (d *floatDomain) next(v, to float64) float64 {
	if d.bits == 32 {
		return float64(math.Nextafter32(float32(v), float32(to)))
	}

	return math.Nextafter(v, to)
}

func parseFloat(constant string) float64 {
	v, err := strconv.ParseFloat(constant, 64)
	if err != nil {
		panic(fmt.Errorf("invalid float constant %q: %w", constant, err))
	}

	return v
}
