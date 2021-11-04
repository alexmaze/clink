package spinner

import (
	"fmt"
	"time"
)

// TermAction vt100 actions
// ref: http://ascii-table.com/ansi-escape-sequences-vt-100.php
type TermAction string

// TermAction action definations...
const (
	TermActionToUpLine    TermAction = "\x1b[A"
	TermActionToLineHead  TermAction = "\r"
	TermActionCleanLine   TermAction = "\033[2K"
	TermActionCleanScreen TermAction = "\033c"

	TermActionHideCursor TermAction = "\033[?25l"
	TermActionShowCursor TermAction = "\033[?25h"
)

// Execute execute TermAction
func (a TermAction) Execute() {
	fmt.Print(a)
}

// spin icon sample frames
var SampleSpinFrames1 = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var SampleSpinFrames2 = []string{"⁎", "*", "⁑"}
var SampleSpinFrames3 = []string{"..  ", " .. ", "  ..", "   .", ".   "}
var SampleSpinFrames4 = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
var SampleSpinFrames5 = []string{"✵", "✴", "✳", "✶", "✷"}

// DefaultSpinFrames default spin icon frames
var DefaultSpinFrames = SampleSpinFrames1

// DefaultMsg default spin msg
var DefaultMsg = "Working..."

// DefaultSpinGap is time between spin frames, control render rate
// ~ 10fps
const DefaultSpinGap = time.Duration(time.Millisecond * 100)
