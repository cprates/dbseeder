package postgres

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/cprates/dbseeder/internal/generator"
)

// TODO: make generators a parameter so that one can easily config. different types of type generators

// baseType is a base type used to wrap other types so that all types can have a default encode which quotes
// values, e.g.: 'abcd'. Any type like numeric types should implement their own encoding method.
type baseType struct {
	OID     OID
	SubType interface {
		GenVal(*ColumnMeta) *Value
	}
}

func (b baseType) Encode(v any) ([]byte, error) {
	if subT, ok := b.SubType.(interface{ Encode(v any) ([]byte, error) }); ok {
		return subT.Encode(v)
	}
	return encodeQuoted(b.OID, v)
}

func encodeQuoted(oid OID, v any) ([]byte, error) {
	wrap, wrappedDefVal := v.(DefValWrapper)
	if wrappedDefVal {
		v = wrap.V
	}

	if v == nil {
		return nullBytes, nil
	}

	b, err := pgValueMap.Encode(uint32(oid), pgtype.TextFormatCode, v, nil)
	if err != nil {
		return nil, err
	}

	// default values are SQL expressions and must not be quoted
	if wrappedDefVal {
		return b, nil
	}

	b = append(quoteBytes, b...)
	b = append(b, byte('\''))

	return b, nil
}

// Cmp compares two values, if it's sub type implements the Cmp method then that is used, otherwise assumes the given
// values are strings and compares them as such.
func (b baseType) Cmp(l, r any) int {
	if subT, ok := b.SubType.(interface{ Cmp(l, r any) int }); ok {
		return subT.Cmp(l, r)
	}

	if l != nil && r != nil {
		return cmp.Compare(l.(string), r.(string))
	}
	if l == nil && r == nil {
		return 0
	}
	if l == nil {
		return -1
	}

	return 1
}

// DefValWrapper wraps default values that are stored as SQL expressions in psqlColumnMeta, so that these are never
// quoted when encoded. Type encoders should use type assertion to look for this type and avoid quoting values when
// applicable.
type DefValWrapper struct {
	V any
}

func (v DefValWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.V)
}

type (
	Type interface {
		Encode(any) ([]byte, error)
		// Cmp compares l and r. l and r can be of two types, the type implementing the interface or string.
		// It can be a string when the value comes from a parsed SQL string which at the moment is loaded to the
		// type as is. In the future this may change and both are always of the type implementing the interface.
		// This also creates weird cases when using ordering operators on values that are not ordered like 'nil' or
		// 'boolean'. To make this right this needs to be split into tow base operations, comparison and order.
		Cmp(l, r any) int
	}

	OID     uint32
	typeMap map[OID]baseType
)

var (
	nullBytes  = []byte("NULL")
	quoteBytes = []byte("'")
)

// Value is a generated value that can be compared and parsed to string.
type Value struct {
	value any
	_type Type
}

func (v Value) Val() any {
	return v.value
}

func (v *Value) SetVal(val any) {
	v.value = val
}

func (v Value) Cmp(l, r any) int {
	return v._type.Cmp(l, r)
}

func (v Value) String() string {
	b, err := v._type.Encode(v.value)
	if err != nil {
		panic(err)
	}

	return string(b)
}

func (v Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

var (
	pgValueMap = pgtype.NewMap()
)

func newTypeMap() typeMap {
	// Full list of OIDs:
	// https://github.com/jackc/pgx/blob/70f7cad2226dc12406b105f8bb5be9c62780aaf7/pgtype/pgtype.go#L15
	return typeMap{
		pgtype.BoolOID:         baseType{pgtype.BoolOID, typeBoolean{}},
		pgtype.TextOID:         baseType{pgtype.TextOID, typeText{maxLen: 20}},
		pgtype.VarcharOID:      baseType{pgtype.VarcharOID, typeVarChar{}},
		pgtype.VarcharArrayOID: baseType{pgtype.VarcharArrayOID, typeVarArrayChar{}},
		pgtype.BPCharOID:       baseType{pgtype.VarcharOID, typeFixedLenChar{}},
		pgtype.BPCharArrayOID:  baseType{pgtype.BPCharArrayOID, typeFixedLenCharArray{}},
		pgtype.DateOID:         baseType{pgtype.DateOID, typeDate{}},
		pgtype.UUIDOID:         baseType{pgtype.UUIDOID, typeUUID{}},
		pgtype.UUIDArrayOID:    baseType{pgtype.UUIDArrayOID, typeUUIDArray{}},
		pgtype.TimestampOID:    baseType{pgtype.TimestampOID, typeTimestamp{}},
		pgtype.TimestamptzOID:  baseType{pgtype.TimestamptzOID, typeTimestamptz{}},
		pgtype.Int2OID:         baseType{pgtype.Int2OID, typeInt2{}},
		pgtype.Int4OID:         baseType{pgtype.Int4OID, typeInt4{}},
		pgtype.Int8OID:         baseType{pgtype.Int8OID, typeInt8{}},
		pgtype.Float4OID:       baseType{pgtype.Float4OID, typeFloat4{}},
		pgtype.Float8OID:       baseType{pgtype.Float8OID, typeFloat8{}},
		pgtype.NumericOID:      baseType{pgtype.NumericOID, typeNumeric{}},
		pgtype.JSONOID:         baseType{pgtype.JSONOID, typeJSON{}},
		pgtype.JSONBOID:        baseType{pgtype.JSONBOID, typeJSONB{}},
	}
}

func (p *typeMap) registerType(oid OID, tp baseType) {
	(*p)[oid] = tp
}

func (p *typeMap) get(oid OID) (baseType, bool) {
	t, ok := (*p)[oid]

	return t, ok
}

func nullOrDefVal(col *ColumnMeta) (*Value, bool) {
	// Give ~5% chances of a null when column is nullable
	if col.Nullable {
		if n := rand.Intn(99); n < 5 {
			return &Value{_type: col.TypeDef}, true
		}
	}

	// Give ~10% chances of a default value when column has one configured
	if col.DefaultVal != nil {
		if n := rand.Intn(99); n < 10 {
			return &Value{
				value: DefValWrapper{V: col.DefaultVal},
				_type: col.TypeDef,
			}, true
		}
	}

	return nil, false
}

func encodeRaw(oid OID, v any) ([]byte, error) {
	wrap, wrappedDefVal := v.(DefValWrapper)
	if wrappedDefVal {
		v = wrap.V
	}

	if v == nil {
		return nullBytes, nil
	}

	b, err := pgValueMap.Encode(uint32(oid), pgtype.TextFormatCode, v, nil)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type typeBoolean struct{}

func (typeBoolean) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenBool(), _type: col.TypeDef}
}

// Cmp does a comparison between bools is binary but here it must return an int so, return 0 when equal or -1 when
// different.
func (c typeBoolean) Cmp(l, r any) int {
	switch {
	case l == nil && r == nil:
		return 0
	case l == nil || r == nil:
		return -1
	}

	if boolOrString(l) == boolOrString(r) {
		return 0
	}

	return -1
}

func boolOrString(v any) bool {
	if dV, isDefVal := v.(DefValWrapper); isDefVal {
		v = dV.V
	}

	tp, _ := pgValueMap.TypeForOID(pgtype.BoolOID)
	if strL, isStr := v.(string); isStr {
		decoded, err := tp.Codec.DecodeValue(pgValueMap, pgtype.BoolOID, pgtype.TextFormatCode, []byte(strL))
		if err != nil {
			panic(err)
		}

		return decoded.(bool)
	}

	return v.(bool)
}

type typeEnum struct {
	Labels []string
}

func (p typeEnum) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenEnum(p.Labels), _type: col.TypeDef}
}

// Generates a string with a size in the range [0, maxLen]
type typeText struct {
	maxLen int
}

func (p typeText) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenString(p.maxLen), _type: col.TypeDef}
}

type typeVarChar struct{}

func (typeVarChar) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenString(int(col.MaxLen.Int32)), _type: col.TypeDef}
}

type typeVarArrayChar struct{}

func (typeVarArrayChar) GenVal(col *ColumnMeta) *Value {
	return &Value{value: "{" + generator.GenString(int(col.MaxLen.Int32)) + "}", _type: col.TypeDef}
}

type typeFixedLenChar struct{}

func (typeFixedLenChar) GenVal(col *ColumnMeta) *Value {
	val := generator.GenString(int(col.MaxLen.Int32))
	lenExpr := fmt.Sprintf("-%ds", col.MaxLen.Int32)
	// add white space padding to fixed length char types
	val = fmt.Sprintf("%"+lenExpr, val)
	return &Value{value: val, _type: col.TypeDef}
}

type typeFixedLenCharArray struct{}

func (typeFixedLenCharArray) GenVal(col *ColumnMeta) *Value {
	val := generator.GenString(int(col.MaxLen.Int32))
	lenExpr := fmt.Sprintf("-%ds", col.MaxLen.Int32)
	// add white space padding to fixed length char types
	val = fmt.Sprintf("{%"+lenExpr+"}", val)
	return &Value{value: val, _type: col.TypeDef}
}

type typeDate struct{}

func (typeDate) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenDate(), _type: col.TypeDef}
}

func (p typeDate) Cmp(l, r any) int {
	return CmpTimes(l, r)
}

type typeUUID struct{}

func (typeUUID) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenUUID(), _type: col.TypeDef}
}

type typeUUIDArray struct{}

func (typeUUIDArray) GenVal(col *ColumnMeta) *Value {
	return &Value{value: "{" + generator.GenUUID() + "}", _type: col.TypeDef}
}

type typeTimestamp struct{}

func (typeTimestamp) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenTimestamp(), _type: col.TypeDef}
}

func (p typeTimestamp) Cmp(l, r any) int {
	return CmpTimes(l, r)
}

type typeTimestamptz struct{}

func (typeTimestamptz) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenTimestamptz(), _type: col.TypeDef}
}

func (p typeTimestamptz) Cmp(l, r any) int {
	return CmpTimes(l, r)
}

type typeInt2 struct{}

func (typeInt2) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenInt16(), _type: col.TypeDef}
}

func (typeInt2) Encode(v any) ([]byte, error) {
	return encodeRaw(pgtype.Int2OID, v)
}

func (p typeInt2) Cmp(l, r any) int {
	return CmpIntN(l, r, 16)
}

type typeInt4 struct{}

func (typeInt4) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenInt32(), _type: col.TypeDef}
}

func (p typeInt4) Cmp(l, r any) int {
	return CmpIntN(l, r, 32)
}

func (typeInt4) Encode(v any) ([]byte, error) {
	return encodeRaw(pgtype.Int4OID, v)
}

type typeInt8 struct{}

func (typeInt8) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenInt64(), _type: col.TypeDef}
}

func (typeInt8) Encode(v any) ([]byte, error) {
	return encodeRaw(pgtype.Int8OID, v)
}

func (p typeInt8) Cmp(l, r any) int {
	return CmpIntN(l, r, 64)
}

type typeFloat4 struct{}

func (typeFloat4) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenFloat32(), _type: col.TypeDef}
}

func (p typeFloat4) Cmp(l, r any) int {
	return CmpFloat32(l, r)
}

func (typeFloat4) Encode(v any) ([]byte, error) {
	return encodeRaw(pgtype.Float4OID, v)
}

type typeFloat8 struct{}

func (typeFloat8) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenFloat64(), _type: col.TypeDef}
}

func (typeFloat8) Encode(v any) ([]byte, error) {
	return encodeRaw(pgtype.Float8OID, v)
}

func (p typeFloat8) Cmp(l, r any) int {
	return CmpFloat64(l, r)
}

type typeNumeric struct{}

func (typeNumeric) GenVal(col *ColumnMeta) *Value {
	precision := 10
	if col.NumericPrecision.Valid {
		precision = int(col.NumericPrecision.Int32)
	}
	scale := 5
	if col.NumericPrecision.Valid {
		scale = int(col.NumericScale.Int32)
	}

	return &Value{value: generator.GenNumeric(precision-scale, scale), _type: col.TypeDef}
}

func (typeNumeric) Encode(v any) ([]byte, error) {
	return encodeRaw(pgtype.NumericOID, v)
}

type typeJSON struct{}

func (typeJSON) GenVal(col *ColumnMeta) *Value {
	return &Value{value: generator.GenJSON(), _type: col.TypeDef}
}

type typeJSONB struct {
	typeJSON
}
