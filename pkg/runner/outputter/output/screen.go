package output

import (
	"strings"

	"github.com/boy-hack/ksubdomain/v2/pkg/core"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

type ScreenOutput struct {
	windowsWidth int
	silent       bool
	onlyDomain   bool  // 修复 Issue #67: 只输出域名
}

// NewScreenOutput 创建屏幕输出器
// 修复 Issue #67: 支持 onlyDomain 参数
func NewScreenOutput(silent bool, onlyDomain ...bool) (*ScreenOutput, error) {
	windowsWidth := core.GetWindowWith()
	s := new(ScreenOutput)
	s.windowsWidth = windowsWidth
	s.silent = silent
	// 支持可选的 onlyDomain 参数 (向后兼容)
	if len(onlyDomain) > 0 {
		s.onlyDomain = onlyDomain[0]
	}
	return s, nil
}

func (s *ScreenOutput) WriteDomainResult(domain result.Result) error {
	var msg string
	
	// 修复 Issue #67: 支持只输出域名模式
	if s.onlyDomain {
		// 只输出域名,不显示 IP 和其他记录
		msg = domain.Subdomain
	} else {
		// 完整输出: 域名 => 记录1 => 记录2
		var domains []string = []string{domain.Subdomain}
		for _, item := range domain.Answers {
			domains = append(domains, item)
		}
		msg = strings.Join(domains, " => ")
	}
	
	if !s.silent {
		// Pad to terminal width to overwrite any progress-bar remnants,
		// but do NOT prefix with \r — that would corrupt piped output
		// (e.g. `ksubdomain ... --od --silent | httpx`).
		screenWidth := s.windowsWidth - len(msg) - 1
		if screenWidth < 0 {
			screenWidth = 0
		}
		gologger.Silentf("%s% *s\n", msg, screenWidth, "")
	} else {
		// silent=true: plain domain-per-line, no control characters.
		// This is the canonical pipe-friendly mode (--od --silent | httpx).
		gologger.Silentf("%s\n", msg)
	}
	return nil
}

func (s *ScreenOutput) Close() error {
	return nil
}
