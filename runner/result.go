package runner

import (
	"bufio"
	"context"
	"ksubdomain/core"
	"ksubdomain/core/gologger"
	"os"
	"strings"
)

func (r *runner) handleResult(ctx context.Context) {
	var isWrite bool = false
	var err error
	var windowsWidth int

	if r.options.Silent {
		windowsWidth = 0
	} else {
		windowsWidth = core.GetWindowWith()
	}

	if r.options.Output != "" {
		isWrite = true
	}
	var foutput *os.File
	if isWrite {
		foutput, err = os.OpenFile(r.options.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			gologger.Errorf("写入结果文件失败：%s\n", err.Error())
		}
	}
	for result := range r.recver {
		var content []string
		content = append(content, result.Subdomain)
		for _, v := range result.Answers {
			content = append(content, v.String())
		}
		msg := strings.Join(content, " => ")

		fontlenth := windowsWidth - len(msg) - 1
		if !r.options.Silent {
			if windowsWidth > 0 && fontlenth > 0 {
				gologger.Silentf("\r%s% *s\n", msg, fontlenth, "")
			} else {
				gologger.Silentf("\r%s\n", msg)
			}
			// 打印一下结果,可以看得更直观
			r.PrintStatus()
		} else {
			gologger.Silentf("%s\n", msg)
		}
		if isWrite {
			w := bufio.NewWriter(foutput)
			_, err = w.WriteString(content[0] + "\n")
			if err != nil {
				gologger.Errorf("写入结果文件失败.Err:%s\n", err.Error())
			}
			_ = w.Flush()
		}
	}
}
