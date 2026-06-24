package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestEncodeQuoted(t *testing.T) {
	_, err := encodeQuoted(pgtype.TextOID, 42)
	require.Error(t, err)
	require.Equal(t, "unable to encode 42 into text format for text (OID 25): cannot find encode plan", err.Error())

	b, err := encodeQuoted(pgtype.TextOID, "")
	require.NoError(t, err)
	require.Equal(t, []byte("''"), b)

	b, err = encodeQuoted(pgtype.TextOID, nil)
	require.NoError(t, err)
	require.Equal(t, []byte("NULL"), b)

	b, err = encodeQuoted(pgtype.TextOID, "qwerty")
	require.NoError(t, err)
	require.Equal(t, []byte("'qwerty'"), b)

	b, err = encodeQuoted(pgtype.Int2OID, 42)
	require.NoError(t, err)
	require.Equal(t, []byte("'42'"), b)

	b, err = encodeQuoted(pgtype.Int2OID, DefValWrapper{V: 42})
	require.NoError(t, err)
	require.Equal(t, []byte("42"), b)

	b, err = encodeQuoted(pgtype.Int2OID, DefValWrapper{V: nil})
	require.NoError(t, err)
	require.Equal(t, []byte("NULL"), b)
}

func TestEncodeRaw(t *testing.T) {
	_, err := encodeRaw(pgtype.TextOID, 42)
	require.Error(t, err)
	require.Equal(t, "unable to encode 42 into text format for text (OID 25): cannot find encode plan", err.Error())

	b, err := encodeRaw(pgtype.TextOID, "")
	require.NoError(t, err)
	require.Equal(t, ([]byte)(nil), b)

	b, err = encodeRaw(pgtype.TextOID, nil)
	require.NoError(t, err)
	require.Equal(t, []byte("NULL"), b)

	b, err = encodeRaw(pgtype.TextOID, "qwerty")
	require.NoError(t, err)
	require.Equal(t, []byte("qwerty"), b)

	b, err = encodeRaw(pgtype.Int2OID, 0)
	require.NoError(t, err)
	require.Equal(t, []byte("0"), b)

	b, err = encodeRaw(pgtype.Int2OID, 42)
	require.NoError(t, err)
	require.Equal(t, []byte("42"), b)

	b, err = encodeRaw(pgtype.Int2OID, DefValWrapper{V: 42})
	require.NoError(t, err)
	require.Equal(t, []byte("42"), b)

	b, err = encodeRaw(pgtype.Int2OID, DefValWrapper{V: nil})
	require.NoError(t, err)
	require.Equal(t, []byte("NULL"), b)
}

func TestTypeGenericCmp(t *testing.T) {
	g := baseType{pgtype.TextOID, typeText{maxLen: 20}}
	require.Equal(t, -1, g.Cmp("abc", "z"))
	require.Equal(t, 1, g.Cmp("z", "abc"))
	require.Equal(t, 0, g.Cmp("z", "z"))
	require.Equal(t, 0, g.Cmp(nil, nil))
	require.Equal(t, -1, g.Cmp(nil, "z"))
	require.Equal(t, 1, g.Cmp("z", nil))

	g = baseType{pgtype.BoolOID, typeBoolean{}}
	require.Equal(t, 0, g.Cmp(true, true))
}

func TestTypeBoolCmp(t *testing.T) {
	b := typeBoolean{}
	require.Equal(t, -1, b.Cmp(false, true))
	require.Equal(t, -1, b.Cmp(true, false))
	require.Equal(t, 0, b.Cmp(false, false))
	require.Equal(t, 0, b.Cmp(true, true))
}

func TestTypeDateCmp(t *testing.T) {
	dt := typeDate{}
	now := time.Now()
	future := time.Now().Add(time.Minute)
	require.Equal(t, -1, dt.Cmp(now, future))
	require.Equal(t, 1, dt.Cmp(future, now))
	require.Equal(t, 0, dt.Cmp(now, now))
}

func TestTypeTimestampCmp(t *testing.T) {
	ts := typeTimestamp{}
	now := time.Now()
	future := time.Now().Add(time.Minute)
	require.Equal(t, -1, ts.Cmp(now, future))
	require.Equal(t, 1, ts.Cmp(future, now))
	require.Equal(t, 0, ts.Cmp(now, now))
}

func TestTypeTimestamptzCmp(t *testing.T) {
	ts := typeTimestamptz{}
	now := time.Now()
	future := time.Now().Add(time.Minute)
	require.Equal(t, -1, ts.Cmp(now, future))
	require.Equal(t, 1, ts.Cmp(future, now))
	require.Equal(t, 0, ts.Cmp(now, now))
}

func TestTypeInt2Cmp(t *testing.T) {
	i := typeInt2{}
	require.Equal(t, -1, i.Cmp(1, 2))
	require.Equal(t, 1, i.Cmp(2, 1))
	require.Equal(t, 0, i.Cmp(1, 1))
	require.Equal(t, 0, i.Cmp(nil, nil))
	require.Equal(t, -1, i.Cmp(nil, 1))
	require.Equal(t, 1, i.Cmp(1, nil))
}

func TestTypeInt4Cmp(t *testing.T) {
	i := typeInt4{}
	require.Equal(t, -1, i.Cmp(1, 2))
	require.Equal(t, 1, i.Cmp(2, 1))
	require.Equal(t, 0, i.Cmp(1, 1))
	require.Equal(t, 0, i.Cmp(nil, nil))
	require.Equal(t, -1, i.Cmp(nil, 1))
	require.Equal(t, 1, i.Cmp(1, nil))
}

func TestTypeInt8Cmp(t *testing.T) {
	i := typeInt8{}
	require.Equal(t, -1, i.Cmp(1, 2))
	require.Equal(t, 1, i.Cmp(2, 1))
	require.Equal(t, 0, i.Cmp(1, 1))
	require.Equal(t, 0, i.Cmp(nil, nil))
	require.Equal(t, -1, i.Cmp(nil, 1))
	require.Equal(t, 1, i.Cmp(1, nil))
}

func TestBoolOrString(t *testing.T) {
	b := boolOrString("t")
	require.True(t, b)
	b = boolOrString("true")
	require.True(t, b)
	b = boolOrString("1")
	require.True(t, b)
	b = boolOrString(true)
	require.True(t, b)

	b = boolOrString("f")
	require.False(t, b)
	b = boolOrString("false")
	require.False(t, b)
	b = boolOrString("0")
	require.False(t, b)
	b = boolOrString(false)
	require.False(t, b)

	b = boolOrString(DefValWrapper{V: true})
	require.True(t, b)
}

func TestTypeFloat4Cmp(t *testing.T) {
	i := typeFloat4{}
	require.Equal(t, -1, i.Cmp(float32(1.42), float32(2.24)))
	require.Equal(t, 1, i.Cmp(float32(2.42), float32(1.24)))
	require.Equal(t, 0, i.Cmp(float32(1.42), float32(1.42)))
	require.Equal(t, 0, i.Cmp(nil, nil))
	require.Equal(t, -1, i.Cmp(nil, float32(1.24)))
	require.Equal(t, 1, i.Cmp(float32(1.42), nil))
}

func TestTypeFloat8Cmp(t *testing.T) {
	i := typeFloat8{}
	require.Equal(t, -1, i.Cmp(1.42, 2.24))
	require.Equal(t, 1, i.Cmp(2.42, 1.24))
	require.Equal(t, 0, i.Cmp(1.42, 1.42))
	require.Equal(t, 0, i.Cmp(nil, nil))
	require.Equal(t, -1, i.Cmp(nil, 1.24))
	require.Equal(t, 1, i.Cmp(1.42, nil))
}

func TestValue(t *testing.T) {
	v := Value{value: "abc", _type: baseType{pgtype.TextOID, typeText{maxLen: 20}}}
	require.Equal(t, 0, v.Cmp("z", "z"))
	require.Equal(t, "abc", v.Val())
	require.Equal(t, "'abc'", v.String())
	str, err := v.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, `"abc"`, string(str))
	v.SetVal("42")
	require.Equal(t, "42", v.Val())
}

func TestTypeFixedLenChar_GenVal(t *testing.T) {
	typ := typeFixedLenChar{}
	for range 1000 {
		val := typ.GenVal(&ColumnMeta{MaxLen: sql.NullInt32{Int32: 3}})
		require.Len(t, val.Val(), 3)
	}
}
