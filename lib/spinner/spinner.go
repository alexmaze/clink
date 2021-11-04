/*Package spinner provider terminal loading line with spin icon & msg

Usage:
	sp := spinner.New().Start()
	sp.SetMsg("Working on 1, 12345678901234567890123456789012345678901234567890123456789012345678901234567890")
	sp.CheckPoint(spinner.IconCheck, spinner.ColorBlue, "looks good for now~", spinner.ColorPurple)
	sp.Success("Everything good!")
*/
package spinner

import (
	"time"

	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/icon"
)

// Spinner show loading line with spin icon & msg
// e.g. "⠋ Loading"
type Spinner interface {
	Start(msg ...string) Spinner                                                        // start spin
	Stop()                                                                              // clean
	CheckPoint(icon icon.Icon, iconColor color.Color, msg string, msgColor color.Color) // set check point line with icon & msg
	Success(msg string)                                                                 // alias to Stop() & CheckPoint(IconCheck, ColorGreen, msg, ColorReset)
	Failed(msg string)                                                                  // alias to Stop() & CheckPoint(IconCross, ColorRed, msg, ColorReset)
	Successf(format string, args ...interface{})                                        // alias to Stop() & CheckPoint(IconCheck, ColorGreen, msg, ColorReset)
	Failedf(format string, args ...interface{})                                         // alias to Stop() & CheckPoint(IconCross, ColorRed, msg, ColorReset)

	SetSpinGap(spinGap time.Duration) Spinner // custom spin render time gap
	SetIconFrames([]string) Spinner           // custom spin icon frames
	SetIconColor(color.Color) Spinner         // custom spin icon color
	SetMsg(msg string) Spinner                // custom spin msg content
	SetMsgColor(color color.Color) Spinner    // custom spin msg color
}
