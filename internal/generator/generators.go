package generator

import (
	"errors"
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

// this is not a linear string generator, not all strings have the same probability. Smaller the string higher the
// probability of being generated.
func genVarString(min, max int, dict []rune) string {
	if min > max {
		panic(fmt.Errorf("min. %d must not be larger than max. %d", min, max))
	}
	if len(dict) == 0 {
		panic(errors.New("dict cannot be empty"))
	}

	maxLen := max - min + 1
	size := min + rand.Intn(maxLen)
	// hack to create a lower probability for strings of size <= 3 in order to reduce collisions which hit hard
	// generated rows throughput when unique indexes are involved. Moving forward random strings may be replaced
	// with something more useful so not going to spend more time here for now.
	if maxLen > 3 {
		for i := 0; size <= 3 && i < 3; i++ {
			size = min + rand.Intn(maxLen)
		}
	}
	// this is a hack to make the empty string to have the same probability as any other character in a single
	// character string
	if size == 0 {
		v := string(dict[rand.Intn(len(dict))])
		if v == "0" {
			return ""
		}

		return v
	}

	src := make([]rune, size)
	for i := range src {
		src[i] = dict[rand.Intn(len(dict))]
	}

	return string(src)
}

func GenBool() bool {
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
