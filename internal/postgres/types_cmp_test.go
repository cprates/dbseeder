package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCmpIntN(t *testing.T) {
	testsSet := []struct {
		description string
		l           any
		r           any
		bits        int
		expect      int
	}{
		{
			description: "nil on the left",
			l:           nil,
			r:           42,
			bits:        64,
			expect:      -1,
		},
		{
			description: "nil on the right",
			l:           42,
			r:           nil,
			bits:        64,
			expect:      1,
		},
		{
			description: "both nil",
			l:           nil,
			r:           nil,
			bits:        64,
			expect:      0,
		},
		{
			description: "l > r",
			l:           42,
			r:           41,
			bits:        64,
			expect:      1,
		},
		{
			description: "l < r",
			l:           42,
			r:           43,
			bits:        64,
			expect:      -1,
		},
		{
			description: "l == r",
			l:           42,
			r:           42,
			bits:        64,
			expect:      0,
		},
		{
			description: "strings get converted",
			l:           "42",
			r:           "41",
			bits:        64,
			expect:      1,
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			r := CmpIntN(test.l, test.r, test.bits)
			require.Equal(t, test.expect, r)
		})
	}
}

func TestCmpFloat64(t *testing.T) {
	testsSet := []struct {
		description string
		l           any
		r           any
		expect      int
	}{
		{
			description: "nil on the left",
			l:           nil,
			r:           42.24,
			expect:      -1,
		},
		{
			description: "nil on the right",
			l:           42.24,
			r:           nil,
			expect:      1,
		},
		{
			description: "both nil",
			l:           nil,
			r:           nil,
			expect:      0,
		},
		{
			description: "l > r",
			l:           42.24,
			r:           41.14,
			expect:      1,
		},
		{
			description: "l < r",
			l:           42.42,
			r:           43.34,
			expect:      -1,
		},
		{
			description: "l == r",
			l:           42.42,
			r:           42.42,
			expect:      0,
		},
		{
			description: "strings get converted",
			l:           "42.42",
			r:           "41.24",
			expect:      1,
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			r := CmpFloat64(test.l, test.r)
			require.Equal(t, test.expect, r)
		})
	}
}

func TestCmpFloat32(t *testing.T) {
	testsSet := []struct {
		description string
		l           any
		r           any
		expect      int
	}{
		{
			description: "nil on the left",
			l:           nil,
			r:           float32(42.24),
			expect:      -1,
		},
		{
			description: "nil on the right",
			l:           float32(42.24),
			r:           nil,
			expect:      1,
		},
		{
			description: "both nil",
			l:           nil,
			r:           nil,
			expect:      0,
		},
		{
			description: "l > r",
			l:           float32(42.24),
			r:           float32(41.14),
			expect:      1,
		},
		{
			description: "l < r",
			l:           float32(42.42),
			r:           float32(43.34),
			expect:      -1,
		},
		{
			description: "l == r",
			l:           float32(42.42),
			r:           float32(42.42),
			expect:      0,
		},
		{
			description: "strings get converted",
			l:           "42.42",
			r:           "41.24",
			expect:      1,
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			r := CmpFloat32(test.l, test.r)
			require.Equal(t, test.expect, r)
		})
	}
}

func TestAnyToTime(t *testing.T) {
	testsSet := []struct {
		description string
		input       any
		expect      time.Time
	}{
		{
			description: "time.Time is returned as is",
			input:       time.Date(2025, 12, 12, 12, 12, 12, 12, time.UTC),
			expect:      time.Date(2025, 12, 12, 12, 12, 12, 12, time.UTC),
		},
		{
			description: "string timestamp is parsed",
			input:       "2025-12-12T12:12:12.000000012Z",
			expect:      time.Date(2025, 12, 12, 12, 12, 12, 12, time.UTC),
		},
		{
			description: "default values are unwrapped",
			input:       DefValWrapper{V: "2025-12-12T12:12:12.000000012Z"},
			expect:      time.Date(2025, 12, 12, 12, 12, 12, 12, time.UTC),
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			tm := anyToTime(test.input)
			require.Equal(t, test.expect, tm)
		})
	}
}

func TestAnyToInt64(t *testing.T) {
	testsSet := []struct {
		description string
		input       any
		bits        int
		expect      int64
	}{
		{
			description: "42 is returned as is",
			input:       42,
			bits:        64,
			expect:      42,
		},
		{
			description: "string int64 is parsed as 64 bits",
			input:       "42",
			bits:        64,
			expect:      42,
		},
		{
			description: "string int64 is parsed as 32 bits",
			input:       "42",
			bits:        32,
			expect:      42,
		},
		{
			description: "default values are unwrapped",
			input:       DefValWrapper{V: "42"},
			bits:        64,
			expect:      42,
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			tm := anyToInt64(test.input, test.bits)
			require.Equal(t, test.expect, tm)
		})
	}
}

func TestAnyToFloat64(t *testing.T) {
	testsSet := []struct {
		description string
		input       any
		expect      float64
	}{
		{
			description: "42.24 is returned as is",
			input:       42.24,
			expect:      42.24,
		},
		{
			description: "string float64 is parsed as 64 bits",
			input:       "42.24",
			expect:      42.24,
		},
		{
			description: "default values are unwrapped",
			input:       DefValWrapper{V: "42.24"},
			expect:      42.24,
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			tm := anyToFloat64(test.input)
			require.Equal(t, test.expect, tm)
		})
	}
}

func TestAnyToFloat32(t *testing.T) {
	testsSet := []struct {
		description string
		input       any
		expect      float32
	}{
		{
			description: "42.24 is returned as is",
			input:       float32(42.24),
			expect:      42.24,
		},
		{
			description: "string float64 is parsed as 64 bits",
			input:       "42.24",
			expect:      42.24,
		},
		{
			description: "default values are unwrapped",
			input:       DefValWrapper{V: "42.24"},
			expect:      42.24,
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			tm := anyToFloat32(test.input)
			require.Equal(t, test.expect, tm)
		})
	}
}

func TestCmpTimes(t *testing.T) {
	testsSet := []struct {
		description string
		l           any
		r           any
		expect      int
	}{
		{
			description: "equal time.Time",
			l:           time.Date(2025, 12, 12, 12, 12, 12, 12, time.UTC),
			r:           time.Date(2025, 12, 12, 12, 12, 12, 12, time.UTC),
			expect:      0,
		},
		{
			description: "equal string timestamps",
			l:           "2025-12-12T12:12:12.000000012Z",
			r:           "2025-12-12T12:12:12.000000012Z",
			expect:      0,
		},
		{
			description: "compare time.Time",
			l:           time.Date(2025, 12, 12, 12, 12, 12, 10, time.UTC),
			r:           time.Date(2025, 12, 12, 12, 12, 12, 12, time.UTC),
			expect:      -1,
		},
		{
			description: "left as nil",
			l:           nil,
			r:           "2025-12-12T12:12:12.000000012Z",
			expect:      -1,
		},
		{
			description: "right as nil",
			l:           "2025-12-12T12:12:12.000000012Z",
			r:           nil,
			expect:      1,
		},
		{
			description: "left and right nil",
			l:           nil,
			r:           nil,
			expect:      0,
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			r := CmpTimes(test.l, test.r)
			require.Equal(t, test.expect, r)
		})
	}
}

func TestDefValWrapperMarshal(t *testing.T) {
	d := DefValWrapper{V: "abc"}
	b, err := d.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, `"abc"`, string(b))
}
