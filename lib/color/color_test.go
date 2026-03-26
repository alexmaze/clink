package color

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColor_Red(t *testing.T) {
	const text = "red"
	assert.Equal(t, string(ColorRed)+text+string(ColorReset), ColorRed.Color(text))
}

func TestColor_Green(t *testing.T) {
	assert.Equal(t, string(ColorGreen)+"ok"+string(ColorReset), ColorGreen.Color("ok"))
}

func TestColor_Yellow(t *testing.T) {
	assert.Equal(t, string(ColorYellow)+"warn"+string(ColorReset), ColorYellow.Color("warn"))
}

func TestColor_Blue(t *testing.T) {
	assert.Equal(t, string(ColorBlue)+"info"+string(ColorReset), ColorBlue.Color("info"))
}

func TestColor_EmptyString(t *testing.T) {
	assert.Equal(t, string(ColorRed)+string(ColorReset), ColorRed.Color(""))
}

func TestColor_AllConstants(t *testing.T) {
	// Verify all color constants are non-empty and contain escape codes
	colors := []Color{
		ColorReset, ColorRed, ColorGreen, ColorYellow,
		ColorBlue, ColorPurple, ColorCyan, ColorGray, ColorWhite,
	}
	for _, c := range colors {
		assert.NotEmpty(t, string(c))
		assert.Contains(t, string(c), "\033[")
	}
}

func TestColor_WithSpecialChars(t *testing.T) {
	result := ColorCyan.Color("hello\nworld")
	assert.Contains(t, result, "hello\nworld")
	assert.Contains(t, result, string(ColorCyan))
	assert.Contains(t, result, string(ColorReset))
}

func TestColor_Chaining(t *testing.T) {
	// Test that colored strings can be concatenated
	result := ColorRed.Color("error") + " " + ColorGreen.Color("success")
	assert.Contains(t, result, "error")
	assert.Contains(t, result, "success")
}
