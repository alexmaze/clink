package spinner

import (
	"fmt"
	"sync"
	"time"

	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/icon"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// spinnerLipglossStyle is the lipgloss style used to colour the spinning frame.
var spinnerLipglossStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true) // bright cyan

// New create a spinner
func New(msg ...string) Spinner {
	m := DefaultMsg
	if len(msg) > 0 {
		m = msg[0]
	}

	return &spinner{
		spinGap:            DefaultSpinGap,
		msg:                m,
		msgColor:           color.ColorReset,
		spinIconFrames:     DefaultSpinFrames,
		spinIconFrameIndex: 0,
		spinIconColor:      color.ColorCyan,
	}
}

// spinner implements the Spinner interface.
type spinner struct {
	ticker *time.Ticker
	lock   sync.Mutex
	done   chan struct{}

	spinGap            time.Duration
	spinIconFrames     []string
	spinIconFrameIndex int
	spinIconColor      color.Color
	msg                string
	msgColor           color.Color
}

func (s *spinner) Start(msg ...string) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	TermActionHideCursor.Execute()

	if len(msg) > 0 {
		s.msg = msg[0]
	}

	s.reset()
	s.run()

	return s
}

func (s *spinner) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()

	TermActionShowCursor.Execute()

	if s.done != nil {
		close(s.done)
		s.done = nil
	}
}

func (s *spinner) CheckPoint(icon icon.Icon, iconColor color.Color, msg string, msgColor color.Color) {
	fmt.Printf("%v%v%v %v\n", TermActionCleanLine, TermActionToLineHead, iconColor.Color(string(icon)), msgColor.Color(msg))
}

func (s *spinner) Success(msg string) {
	s.Stop()
	s.CheckPoint(icon.IconCheck, color.ColorGreen, msg, color.ColorReset)
}

func (s *spinner) Failed(msg string) {
	s.Stop()
	s.CheckPoint(icon.IconCross, color.ColorRed, msg, color.ColorReset)
}

func (s *spinner) Successf(format string, args ...interface{}) {
	s.Success(fmt.Sprintf(format, args...))
}

func (s *spinner) Failedf(format string, args ...interface{}) {
	s.Failed(fmt.Sprintf(format, args...))
}

func (s *spinner) SetMsg(msg string) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.msg = msg

	return s
}

func (s *spinner) SetMsgColor(color color.Color) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.msgColor = color

	return s
}

func (s *spinner) SetIconFrames(frames []string) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.spinIconFrames = frames

	return s
}

func (s *spinner) SetIconColor(color color.Color) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.spinIconColor = color

	return s
}

func (s *spinner) SetSpinGap(spinGap time.Duration) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	if spinGap != s.spinGap {
		s.spinGap = spinGap
		s.ticker.Reset(spinGap)
	}

	return s
}

func (s *spinner) reset() {
	// Stop any existing ticker.
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.ticker = time.NewTicker(s.spinGap)
	s.ticker.Stop()

	// Signal any existing goroutine to stop, then allocate a fresh channel.
	if s.done != nil {
		close(s.done)
	}
	s.done = make(chan struct{})
}

func (s *spinner) run() {
	s.ticker.Reset(s.spinGap)

	// Capture the current done channel so the goroutine holds a stable
	// reference; the struct field may be replaced by a subsequent reset().
	doneCh := s.done

	go func() {
		for {
			select {
			case <-doneCh:
				return
			case <-s.ticker.C:
				s.render()
			}
		}
	}()
}

func (s *spinner) render() {
	// Use a full write lock because newFrame() mutates spinIconFrameIndex.
	s.lock.Lock()
	defer s.lock.Unlock()

	frame := s.newFrame()

	// Resize frame content to fit in a single terminal line.
	w, _, err := term.GetSize(0)
	if err == nil {
		if len(frame) > w {
			frame = frame[0:(w-3)] + "..."
		}
	}

	fmt.Printf("%v%v%v", TermActionCleanLine, TermActionToLineHead, frame)
}

func (s *spinner) newFrame() string {
	s.spinIconFrameIndex++
	if s.spinIconFrameIndex >= len(s.spinIconFrames) {
		s.spinIconFrameIndex = 0
	}

	spinIconFrame := s.spinIconFrames[s.spinIconFrameIndex]
	// Use lipgloss for the spinner icon; keep legacy color.Color for the message.
	return fmt.Sprintf("%s %s", spinnerLipglossStyle.Render(spinIconFrame), s.msgColor.Color(s.msg))
}
