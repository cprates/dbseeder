package generator

// This file is a collection of value generators for different golang types.

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	strChars  = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	strDigits = []rune("0123456789")
)

func genVarString(min, max int, dict []rune) string {
	if min > max {
		panic(fmt.Errorf("min. %d must not be larger than max. %d", min, max))
	}

	size := min + rand.Intn(max-min+1)
	b := make([]rune, size)
	for i := range b {
		b[i] = dict[rand.Intn(len(dict))]
	}
	return string(b)
}

func GenBool() bool {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	return rand.Intn(2) == 0
}

func GenString(max int) string {
	return genVarString(0, max, strChars)
}

func GenUUID() string {
	return uuid.New().String()
}

func GenTimestamp() time.Time {
	return time.Now().UTC()
}

func GenTimestamptz() time.Time {
	return time.Now()
}

func GenDate() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func GenInt16() int16 {
	return int16(rand.Uint32() & (1<<16 - 1))
}

func GenInt32() int32 {
	return rand.Int31()
}

func GenInt64() int64 {
	return rand.Int63()
}

func GenFloat32() float32 {
	return rand.Float32()
}

func GenFloat64() float64 {
	return rand.Float64()
}

func GenNumeric(maxLeft, maxRight int) string {
	// use string to avoid rounding problems
	var left string
	if maxLeft > 0 {
		left = genVarString(1, rand.Intn(maxLeft)+1, strDigits)
		// avoid generating number that as string are different but numerically are equivalent, e.g.: 03.3 or 3.30
		left = strings.TrimLeft(left, "0")
	}
	if left == "" {
		left = "0"
	}
	var right string
	if maxRight > 0 {
		right = genVarString(1, rand.Intn(maxRight)+1, strDigits)
		right = strings.TrimRight(right, "0")
	}
	if right == "" {
		right = "0"
	}
	return left + "." + right
}

func GenJSON() string {
	return fmt.Sprintf(`{"numeric_field0":%d}`, rand.Int31())
}

func GenEnum(labels []string) string {
	return labels[rand.Intn(len(labels))]
}
