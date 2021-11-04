package color

import "fmt"

// Color color enum
type Color string

// Colors colors definations
// ref: https://twinnation.org/articles/35/how-to-add-colors-to-your-console-terminal-output-in-go
const (
	ColorReset Color = "\033[0m"

	ColorRed    Color = "\033[31m"
	ColorGreen  Color = "\033[32m"
	ColorYellow Color = "\033[33m"
	ColorBlue   Color = "\033[34m"
	ColorPurple Color = "\033[35m"
	ColorCyan   Color = "\033[36m"
	ColorGray   Color = "\033[37m"
	ColorWhite  Color = "\033[97m"
)

// Color color the string in terminal
func (c Color) Color(str string) string {
	return fmt.Sprintf("%s%s%s", c, str, ColorReset)
}
