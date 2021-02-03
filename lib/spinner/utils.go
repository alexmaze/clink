package spinner

// TermAction vt100 actions
// ref: http://ascii-table.com/ansi-escape-sequences-vt-100.php
type TermAction string

// TermAction action definations...
const (
	TermActionToUpLine    = "\x1b[A"
	TermActionToLineHead  = "\r"
	TermActionCleanLine   = "\033[2K"
	TermActionCleanScreen = "\033c"
)

// spin icon sample frames
var sampleSpinFrames1 = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var sampleSpinFrames2 = []string{"⁎", "*", "⁑"}
var sampleSpinFrames3 = []string{".  ", " . ", "  ."}
var sampleSpinFrames4 = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}

// DefaultSpinFrames default spin icon frames
var DefaultSpinFrames = sampleSpinFrames1

// DefaultMsg default spin msg
var DefaultMsg = "Working..."
