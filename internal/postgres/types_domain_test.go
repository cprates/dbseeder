package postgres

import (
	"database/sql"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func newTestColumn(oid OID) *ColumnMeta {
	types := newTypeMap()
	tp, ok := types.get(oid)
	if !ok {
		panic("unknown type")
	}

	return &ColumnMeta{Name: "c1", TypeDef: tp}
}

func TestIntDomainExcludedForDomainIsReset(t *testing.T) {
	col := newTestColumn(pgtype.Int8OID)
	d := col.NewDomain().(*intDomain)

	d.excludedForDomain = true
	d.Above("0", true)
	require.False(t, d.excludedForDomain)

	d.excludedForDomain = true
	d.Below("2", true)
	require.False(t, d.excludedForDomain)

	d.excludedForDomain = true
	d.Exclude("1")
	require.False(t, d.excludedForDomain)
}

func TestSupportsDomain(t *testing.T) {
	for _, oid := range []OID{
		pgtype.Int2OID, pgtype.Int4OID, pgtype.Int8OID, pgtype.Float4OID, pgtype.Float8OID, pgtype.NumericOID,
		pgtype.DateOID, pgtype.TimestampOID, pgtype.TimestamptzOID,
	} {
		require.True(t, newTestColumn(oid).SupportsDomain(), "oid %d", oid)
	}

	for _, oid := range []OID{pgtype.TextOID, pgtype.VarcharOID, pgtype.BoolOID, pgtype.UUIDOID} {
		require.False(t, newTestColumn(oid).SupportsDomain(), "oid %d", oid)
		require.Panics(t, func() { newTestColumn(oid).NewDomain() })
	}
}

func TestIntDomain(t *testing.T) {
	testsSet := []struct {
		description string
		oid         OID
		build       func(d Domain)
		// expected inclusive range and excluded values
		lo, hi   int64
		excluded []int64
	}{
		{
			description: "exclusive bounds",
			oid:         pgtype.Int4OID,
			build: func(d Domain) {
				d.Above("42", false)
				d.Below("50", false)
			},
			lo: 43, hi: 49,
		},
		{
			description: "inclusive bounds",
			oid:         pgtype.Int4OID,
			build: func(d Domain) {
				d.Above("42", true)
				d.Below("50", true)
			},
			lo: 42, hi: 50,
		},
		{
			description: "tightest bounds win",
			oid:         pgtype.Int8OID,
			build: func(d Domain) {
				d.Above("10", false)
				d.Above("42", false)
				d.Above("20", true)
				d.Below("100", true)
				d.Below("50", false)
				d.Below("70", false)
			},
			lo: 43, hi: 49,
		},
		{
			description: "negative bounds",
			oid:         pgtype.Int2OID,
			build: func(d Domain) {
				d.Above("-10", true)
				d.Below("-5", false)
			},
			lo: -10, hi: -6,
		},
		{
			description: "fractional constants",
			oid:         pgtype.Int4OID,
			build: func(d Domain) {
				d.Above("42.5", true)
				d.Below("50.5", false)
			},
			lo: 43, hi: 50,
		},
		{
			description: "negative fractional constants",
			oid:         pgtype.Int4OID,
			build: func(d Domain) {
				d.Above("-5.5", false)
				d.Below("-1.5e-1", true)
			},
			lo: -5, hi: -1,
		},
		{
			description: "exclusions",
			oid:         pgtype.Int4OID,
			build: func(d Domain) {
				d.Above("42", false)
				d.Below("50", true)
				d.Exclude("43")
				d.Exclude("50")
				d.Exclude("45")
				d.Exclude("45")
				d.Exclude("100")
				d.Exclude("46.5")
			},
			lo: 43, hi: 50, excluded: []int64{43, 45, 50},
		},
		{
			description: "single value left",
			oid:         pgtype.Int4OID,
			build: func(d Domain) {
				d.Above("42", false)
				d.Below("45", false)
				d.Exclude("44")
			},
			lo: 43, hi: 44, excluded: []int64{44},
		},
		{
			description: "only lower bound",
			oid:         pgtype.Int2OID,
			build: func(d Domain) {
				d.Above("32760", false)
			},
			lo: 32761, hi: math.MaxInt16,
		},
		{
			description: "only upper bound",
			oid:         pgtype.Int8OID,
			build: func(d Domain) {
				d.Below("-9223372036854775800", true)
			},
			lo: math.MinInt64, hi: -9223372036854775800,
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			col := newTestColumn(test.oid)
			d := col.NewDomain()
			test.build(d)

			for range 1000 {
				n := anyToInt64(d.Gen(), 64)
				require.GreaterOrEqual(t, n, test.lo)
				require.LessOrEqual(t, n, test.hi)
				require.NotContains(t, test.excluded, n)
			}
		})
	}
}

func TestIntDomainPanicsWhenOutOfBounds(t *testing.T) {
	col := newTestColumn(pgtype.Int8OID)
	d := col.NewDomain()
	require.PanicsWithError(t, "-99999999999999999999 does not fit in an int64", func() {
		d.Above("-99999999999999999999", false)
	})
	require.PanicsWithError(t, "99999999999999999999 does not fit in an int64", func() {
		d.Below("99999999999999999999", false)
	})
}

func TestIntDomainGenReturnsColumnType(t *testing.T) {
	require.IsType(t, int16(0), newTestColumn(pgtype.Int2OID).NewDomain().Gen())
	require.IsType(t, int32(0), newTestColumn(pgtype.Int4OID).NewDomain().Gen())
	require.IsType(t, int64(0), newTestColumn(pgtype.Int8OID).NewDomain().Gen())
}

func TestIntDomainCoversWholeRange(t *testing.T) {
	d := newTestColumn(pgtype.Int4OID).NewDomain()
	d.Above("0", false)
	d.Below("6", true)
	d.Exclude("3")

	observed := map[int32]struct{}{}
	for range 1000 {
		observed[d.Gen().(int32)] = struct{}{}
	}
	require.Equal(t, map[int32]struct{}{1: {}, 2: {}, 4: {}, 5: {}, 6: {}}, observed)
}

func TestNumericDomain(t *testing.T) {
	col := newTestColumn(pgtype.NumericOID)
	col.NumericPrecision = sql.NullInt32{Int32: 5, Valid: true}
	col.NumericScale = sql.NullInt32{Int32: 2, Valid: true}

	t.Run("bounds are rounded to the scale", func(t *testing.T) {
		d := col.NewDomain()
		d.Above("1.2345", false)
		d.Below("1.2701", true)
		d.Exclude("1.25")
		d.Exclude("1.255")

		observed := map[string]struct{}{}
		for range 1000 {
			observed[d.Gen().(string)] = struct{}{}
		}
		require.Equal(t, map[string]struct{}{"1.24": {}, "1.26": {}, "1.27": {}}, observed)
	})

	t.Run("generated values are within bounds", func(t *testing.T) {
		d := col.NewDomain()
		d.Above("-3", true)
		d.Below("2.5", true)
		d.Exclude("0")

		for range 1000 {
			v := d.Gen().(string)
			f := anyToFloat64(v)
			require.GreaterOrEqual(t, f, -3.0)
			require.LessOrEqual(t, f, 2.5)
			require.NotEqual(t, "0", v)
		}
	})

	t.Run("type limits", func(t *testing.T) {
		d := col.NewDomain()
		d.Above("999.98", false)

		require.Equal(t, "999.99", d.Gen())
	})
}

func TestNumericConvFromInt(t *testing.T) {
	require.Equal(t, "0", numericConv{scale: 2}.fromInt(0))
	require.Equal(t, "0.05", numericConv{scale: 2}.fromInt(5))
	require.Equal(t, "-0.5", numericConv{scale: 2}.fromInt(-50))
	require.Equal(t, "123.45", numericConv{scale: 2}.fromInt(12345))
	require.Equal(t, "-123", numericConv{scale: 2}.fromInt(-12300))
	require.Equal(t, "42", numericConv{scale: 0}.fromInt(42))
}

func TestFloatDomain(t *testing.T) {
	for _, oid := range []OID{pgtype.Float4OID, pgtype.Float8OID} {
		t.Run("bounds", func(t *testing.T) {
			d := newTestColumn(oid).NewDomain()
			d.Above("42", false)
			d.Below("123", true)
			d.Exclude("100")

			for range 1000 {
				v := anyToFloat64Any(d.Gen())
				require.Greater(t, v, 42.0)
				require.LessOrEqual(t, v, 123.0)
				require.NotEqual(t, 100.0, v)
			}
		})

		t.Run("unbounded", func(t *testing.T) {
			d := newTestColumn(oid).NewDomain()
			d.Below("-1e10", false)

			for range 1000 {
				v := anyToFloat64Any(d.Gen())
				require.Less(t, v, -1e10)
				require.False(t, math.IsInf(v, 0) || math.IsNaN(v))
			}
		})
	}

	t.Run("single value left", func(t *testing.T) {
		d := newTestColumn(pgtype.Float4OID).NewDomain()
		lo := float32(1.5)
		next := math.Nextafter32(lo, 2)
		d.Above("1.5", true)
		d.Below("1.5000001", true)

		for range 100 {
			v := d.Gen().(float32)
			require.True(t, v == lo || v == next, "%v", v)
		}
	})

	t.Run("excluded bound", func(t *testing.T) {
		d := newTestColumn(pgtype.Float8OID).NewDomain()
		d.Above("1", true)
		d.Below("1", true)
		d.Exclude("1")
		// impossible constraints are not supported but, make sure the excluded value is not generated
		require.NotEqual(t, 1.0, d.Gen())
	})
}

func anyToFloat64Any(v any) float64 {
	if f, ok := v.(float32); ok {
		return float64(f)
	}

	return v.(float64)
}

func TestDateDomain(t *testing.T) {
	d := newTestColumn(pgtype.DateOID).NewDomain()
	d.Above("2024-01-01", false)
	d.Below("2024-01-05 12:00:00", true)
	d.Exclude("2024-01-03")

	observed := map[string]struct{}{}
	for range 1000 {
		v := d.Gen().(time.Time)
		require.Zero(t, v.Hour()+v.Minute()+v.Second()+v.Nanosecond())
		observed[v.Format(time.DateOnly)] = struct{}{}
	}
	require.Equal(t, map[string]struct{}{"2024-01-02": {}, "2024-01-04": {}, "2024-01-05": {}}, observed)
}

func TestTimestampDomain(t *testing.T) {
	for _, oid := range []OID{pgtype.TimestampOID, pgtype.TimestamptzOID} {
		d := newTestColumn(oid).NewDomain()
		d.Above("2024-01-01 10:00:00", false)
		d.Below("2024-01-01T10:00:00.000003Z", false)
		d.Exclude("2024-01-01 10:00:00.000001")

		for range 100 {
			v := d.Gen().(time.Time)
			require.Equal(t, time.Date(2024, 1, 1, 10, 0, 0, 2000, time.UTC), v)
		}

		d = newTestColumn(oid).NewDomain()
		d.Above("2024-01-01", true)
		for range 1000 {
			v := d.Gen().(time.Time)
			require.False(t, v.Before(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))
			require.False(t, v.After(time.Now()))
			require.Zero(t, v.Nanosecond()%1000)
		}
	}
}

func TestTimeDomainDefaultUpperBound(t *testing.T) {
	for _, oid := range []OID{pgtype.DateOID, pgtype.TimestampOID, pgtype.TimestamptzOID} {
		t.Run("no bounds are capped at now", func(t *testing.T) {
			d := newTestColumn(oid).NewDomain()
			for range 1000 {
				require.False(t, d.Gen().(time.Time).After(time.Now()))
			}
		})

		t.Run("explicit upper bound replaces now", func(t *testing.T) {
			d := newTestColumn(oid).NewDomain()
			d.Above("2099-01-01", true)
			d.Below("2099-01-03", true)
			for range 1000 {
				v := d.Gen().(time.Time)
				require.False(t, v.Before(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
				require.False(t, v.After(time.Date(2099, 1, 3, 0, 0, 0, 0, time.UTC)))
			}
		})

		t.Run("explicit upper bound beyond now", func(t *testing.T) {
			d := newTestColumn(oid).NewDomain()
			d.Below("2099-01-01", true)
			observedFuture := false
			for range 1000 {
				v := d.Gen().(time.Time)
				require.False(t, v.After(time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)))
				observedFuture = observedFuture || v.After(time.Now())
			}
			require.True(t, observedFuture)
		})

		t.Run("lower bound in the future ignores now", func(t *testing.T) {
			d := newTestColumn(oid).NewDomain()
			d.Above("2099-01-01", false)
			for range 1000 {
				v := d.Gen().(time.Time)
				require.True(t, v.After(time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)))
				require.Less(t, v.Year(), 10000)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	expected := time.Date(2026, 10, 27, 3, 4, 5, 600000000, time.UTC)
	require.True(t, expected.Equal(parseTime("2026-10-27T03:04:05.6Z")))
	require.True(t, expected.Equal(parseTime("2026-10-27 03:04:05.6")))
	require.True(t, expected.Equal(parseTime("2026-10-27 04:04:05.6+01")))
	require.True(t, expected.Equal(parseTime("2026-10-27 04:04:05.6+01:00")))
	require.True(t, time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC).Equal(parseTime("2026-10-27")))
	require.Panics(t, func() { parseTime("not a date") })
}
