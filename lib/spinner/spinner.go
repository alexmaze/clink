/*Package spinner provider terminal loading line with spin icon & msg

Usage:
	sp := spinner.New().Start()
	sp.SetMsg("Working on 1, 12345678901234567890123456789012345678901234567890123456789012345678901234567890")
	sp.CheckPoint(spinner.IconCheck, spinner.ColorBlue, "looks good for now~", spinner.ColorPurple)
	sp.Success("Everything good!")
*/
package spinner

import "fmt"

// Spinner show loading line with spin icon & msg
// e.g. "⠋ Loading"
type Spinner interface {
	Start(msg ...string) Spinner                                       // start spin
	Stop()                                                             // clean
	CheckPoint(icon Icon, iconColor Color, msg string, msgColor Color) // set check point line with icon & msg
	Success(msg string)                                                // alias to Stop() & CheckPoint(IconCheck, ColorGreen, msg, ColorReset)
	Failed(msg string)                                                 // alias to Stop() & CheckPoint(IconCross, ColorRed, msg, ColorReset)

	SetIconFrames([]string) Spinner  // custom spin icon frames
	SetIconColor(Color) Spinner      // custom spin icon color
	SetMsg(msg string) Spinner       // custom spin msg content
	SetMsgColor(color Color) Spinner // custom spin msg color
}

// Color color enum
type Color string

// Colors colors definations
// ref: https://twinnation.org/articles/35/how-to-add-colors-to-your-console-terminal-output-in-go
const (
	ColorReset  Color = "\033[0m"
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
func (c *Color) Color(str string) string {
	return fmt.Sprintf("%s%s%s", *c, str, ColorReset)
}

// Icon used for check point
type Icon string

// Text Icons
const (
	IconCheck Icon = "✔"
	IconCross Icon = "✘"
)
