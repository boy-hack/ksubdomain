package runner

import (
	"bufio"
	"context"
	"errors"
	"github.com/boy-hack/ksubdomain/core"
	"github.com/boy-hack/ksubdomain/core/gologger"
	"github.com/google/gopacket/layers"
	"os"
	"strings"
)

func dnsRecord2String(rr layers.DNSResourceRecord) (string, error) {
	if rr.Class == layers.DNSClassIN {
		switch rr.Type {
		case layers.DNSTypeA, layers.DNSTypeAAAA:
			if rr.IP != nil {
				return rr.IP.String(), nil
			}
		case layers.DNSTypeNS:
			if rr.NS != nil {
				return "NS " + string(rr.NS), nil
			}
		case layers.DNSTypeCNAME:
			if rr.CNAME != nil {
				return "CNAME " + string(rr.CNAME), nil
			}
		case layers.DNSTypePTR:
			if rr.PTR != nil {
				return "PTR " + string(rr.PTR), nil
			}
		case layers.DNSTypeTXT:
			if rr.TXT != nil {
				return "TXT " + string(rr.TXT), nil
			}
		}
	}
	return "", errors.New("dns record error")
}
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
	onlyDomain := r.options.OnlyDomain
	notPrint := r.options.NotPrint
	for result := range r.recver {
		var content []string
		var msg string
		content = append(content, result.Subdomain)

		if onlyDomain {
			msg = result.Subdomain
		} else {
			for _, v := range result.Answers {
				answer, err := dnsRecord2String(v)
				if err != nil {
					continue
				}
				content = append(content, answer)
			}
			msg = strings.Join(content, " => ")
		}

		if !notPrint {
			screenWidth := windowsWidth - len(msg) - 1
			if !r.options.Silent {
				if windowsWidth > 0 && screenWidth > 0 {
					gologger.Silentf("\r%s% *s\n", msg, screenWidth, "")
				} else {
					gologger.Silentf("\r%s\n", msg)
				}
				// 打印一下结果,可以看得更直观
				r.PrintStatus()
			} else {
				gologger.Silentf("%s\n", msg)
			}
		}

		if isWrite {
			w := bufio.NewWriter(foutput)
			_, err = w.WriteString(msg + "\n")
			if err != nil {
				gologger.Errorf("写入结果文件失败.Err:%s\n", err.Error())
			}
			_ = w.Flush()
		}
	}
}
