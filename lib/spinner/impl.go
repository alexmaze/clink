package spinner

import (
	"fmt"
	"sync"
	"time"

	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/icon"
	"golang.org/x/term"
)

// New create a spinner
func New(msg ...string) Spinner {
	m := DefaultMsg
	if len(msg) > 0 {
		m = msg[0]
	}

	return &_Spinner{
		spinGap:            DefaultSpinGap,
		msg:                m,
		msgColor:           color.ColorReset,
		spinIconFrames:     DefaultSpinFrames,
		spinIconFrameIndex: 0,
		spinIconColor:      color.ColorCyan,
	}
}

// _Spinner
// @impl Spinner
type _Spinner struct {
	ticker *time.Ticker
	lock   sync.RWMutex
	done   *chan bool

	spinGap            time.Duration
	spinIconFrames     []string
	spinIconFrameIndex int
	spinIconColor      color.Color
	msg                string
	msgColor           color.Color
}

func (s *_Spinner) Start(msg ...string) Spinner {
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

func (s *_Spinner) Stop() {
	s.lock.RLock()
	defer s.lock.RUnlock()

	TermActionShowCursor.Execute()

	if s.done != nil {
		*s.done <- true
		s.done = nil
	}
}

func (s *_Spinner) CheckPoint(icon icon.Icon, iconColor color.Color, msg string, msgColor color.Color) {
	fmt.Printf("%v%v%v %v\n", TermActionCleanLine, TermActionToLineHead, iconColor.Color(string(icon)), msgColor.Color(msg))
}

func (s *_Spinner) Success(msg string) {
	s.Stop()
	s.CheckPoint(icon.IconCheck, color.ColorGreen, msg, color.ColorReset)
}

func (s *_Spinner) Failed(msg string) {
	s.Stop()
	s.CheckPoint(icon.IconCross, color.ColorRed, msg, color.ColorReset)
}

func (s *_Spinner) Successf(format string, args ...interface{}) {
	s.Success(fmt.Sprintf(format, args...))
}

func (s *_Spinner) Failedf(format string, args ...interface{}) {
	s.Failed(fmt.Sprintf(format, args...))
}

func (s *_Spinner) SetMsg(msg string) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.msg = msg

	return s
}

func (s *_Spinner) SetMsgColor(color color.Color) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.msgColor = color

	return s
}

func (s *_Spinner) SetIconFrames(frames []string) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.spinIconFrames = frames

	return s
}

func (s *_Spinner) SetIconColor(color color.Color) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.spinIconColor = color

	return s
}

func (s *_Spinner) SetSpinGap(spinGap time.Duration) Spinner {
	s.lock.Lock()
	defer s.lock.Unlock()

	if spinGap != s.spinGap {
		s.spinGap = spinGap
		s.ticker.Reset(spinGap)
	}

	return s
}

func (s *_Spinner) reset() {
	// reset ticker
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.ticker = time.NewTicker(s.spinGap)
	s.ticker.Stop()

	// reset done chan
	if s.done != nil {
		*s.done <- true
	}
	newDone := make(chan bool)
	s.done = &newDone
}

func (s *_Spinner) run() {
	s.ticker.Reset(s.spinGap)

	// 启动
	go func() {
		for {
			select {
			case <-*s.done:
				s.done = nil
				return
			case <-s.ticker.C:
				s.render()
			}
		}

	}()
}

func (s *_Spinner) render() {
	s.lock.RLock()
	defer s.lock.RUnlock()

	frame := s.newFrame()

	// resize frame content to fit in single line
	w, _, err := term.GetSize(0)
	if err == nil {
		if len(frame) > w {
			frame = frame[0:(w-3)] + "..."
		}
	}

	fmt.Printf("%v%v%v", TermActionCleanLine, TermActionToLineHead, frame)
}

func (s *_Spinner) newFrame() string {
	s.spinIconFrameIndex++
	if s.spinIconFrameIndex >= len(s.spinIconFrames) {
		s.spinIconFrameIndex = 0
	}

	spinIconFrame := s.spinIconFrames[s.spinIconFrameIndex]
	return fmt.Sprintf("%s %s", s.spinIconColor.Color(spinIconFrame), s.msgColor.Color(s.msg))
}
