package postgres

import (
	"cmp"
	"fmt"
	"strconv"
	"time"
)

func CmpTimes(l, r any) int {
	if r, hasNils := compareNils(l, r); hasNils {
		return r
	}

	left := anyToTime(l)
	right := anyToTime(r)

	return left.Compare(right)
}

func anyToTime(a any) time.Time {
	if dV, isDefVal := a.(DefValWrapper); isDefVal {
		a = dV.V
	}

	switch aT := a.(type) {
	case string:
		parsedA, err := time.Parse(time.RFC3339Nano, aT)
		if err != nil {
			panic(err)
		}
		return parsedA
	case time.Time:
		return aT
	default:
		panic(fmt.Errorf("unexpected type %T", a))
	}
}

func CmpIntN(l, r any, bits int) int {
	if r, hasNils := compareNils(l, r); hasNils {
		return r
	}

	parsedL := anyToInt64(l, bits)
	parsedR := anyToInt64(r, bits)

	return cmp.Compare(parsedL, parsedR)
}

func anyToInt64(n any, bits int) int64 {
	if dV, isDefVal := n.(DefValWrapper); isDefVal {
		n = dV.V
	}

	switch nT := n.(type) {
	case string:
		parsedN, err := strconv.ParseInt(nT, 10, bits)
		if err != nil {
			panic(err)
		}
		return parsedN
	case int:
		return int64(nT)
	case int16:
		return int64(nT)
	case int32:
		return int64(nT)
	case int64:
		return nT
	default:
		panic(fmt.Errorf("unexpected type %T", n))
	}
}

func CmpFloat64(l, r any) int {
	if r, hasNils := compareNils(l, r); hasNils {
		return r
	}

	parsedL := anyToFloat64(l)
	parsedR := anyToFloat64(r)

	return cmp.Compare(parsedL, parsedR)
}

func CmpFloat32(l, r any) int {
	if r, hasNils := compareNils(l, r); hasNils {
		return r
	}

	parsedL := anyToFloat32(l)
	parsedR := anyToFloat32(r)

	return cmp.Compare(parsedL, parsedR)
}

func anyToFloat64(n any) float64 {
	if dV, isDefVal := n.(DefValWrapper); isDefVal {
		n = dV.V
	}

	switch nT := n.(type) {
	case string:
		parsedN, err := strconv.ParseFloat(nT, 64)
		if err != nil {
			panic(err)
		}
		return parsedN
	case float64:
		return nT
	default:
		panic(fmt.Errorf("unexpected type %T", n))
	}
}

func anyToFloat32(n any) float32 {
	if dV, isDefVal := n.(DefValWrapper); isDefVal {
		n = dV.V
	}

	switch nT := n.(type) {
	case string:
		parsedN, err := strconv.ParseFloat(nT, 32)
		if err != nil {
			panic(err)
		}
		return float32(parsedN)
	case float32:
		return nT
	default:
		panic(fmt.Errorf("unexpected type %T", n))
	}
}

func compareNils(l, r any) (int, bool) {
	switch {
	case l == nil && r == nil:
		return 0, true
	case l == nil:
		return -1, true
	case r == nil:
		return 1, true
	}

	return 0, false
}
