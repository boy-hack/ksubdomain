package outputter

import (
	"github.com/boy-hack/ksubdomain/core"
	"github.com/boy-hack/ksubdomain/core/gologger"
)

type ScreenOutput struct {
	windowsWidth int
}

func NewScreenOutput() (*ScreenOutput, error) {
	windowsWidth := core.GetWindowWith()
	s := new(ScreenOutput)
	s.windowsWidth = windowsWidth
	return s, nil
}
func (s *ScreenOutput) Print(msg string) {
	screenWidth := s.windowsWidth - len(msg) - 1
	if s.windowsWidth > 0 && screenWidth > 0 {
		gologger.Silentf("\r%s% *s\n", msg, screenWidth, "")
	} else {
		gologger.Silentf("\r%s\n", msg)
	}
}
