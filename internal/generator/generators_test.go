package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenNumeric(t *testing.T) {
	r := GenNumeric(3, 0)
	require.True(t, strings.HasSuffix(r, ".0"))
	r = GenNumeric(0, 3)
	require.True(t, strings.HasPrefix(r, "0."))
}
