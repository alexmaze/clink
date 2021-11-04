package color

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Color(t *testing.T) {
	const text = "red"

	assert.Equal(t, string(ColorRed+text+ColorReset), ColorRed.Color(text))
}
