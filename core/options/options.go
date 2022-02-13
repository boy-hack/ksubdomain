package options

import (
	"ksubdomain/core"
	"ksubdomain/core/gologger"
	"os"
	"strconv"
)

type Options struct {
	Rate         int64
	Domain       []string
	FileName     string // 字典文件名
	Resolvers    []string
	Output       string // 输出文件名
	Silent       bool
	Stdin        bool
	SkipWildCard bool
	TimeOut      int
	Retry        int
	Method       string // verify模式 enum模式 test模式
	OnlyDomain   bool
	NotPrint     bool
	Level        int
	LevelDomains []string
}

func Band2Rate(bandWith string) int64 {
	suffix := string(bandWith[len(bandWith)-1])
	rate, _ := strconv.ParseInt(string(bandWith[0:len(bandWith)-1]), 10, 64)
	switch suffix {
	case "G":
		fallthrough
	case "g":
		rate *= 1000000000
	case "M":
		fallthrough
	case "m":
		rate *= 1000000
	case "K":
		fallthrough
	case "k":
		rate *= 1000
	default:
		gologger.Fatalf("unknown bandwith suffix '%s' (supported suffixes are G,M and K)\n", suffix)
	}
	packSize := int64(80) // 一个DNS包大概有74byte
	rate = rate / packSize
	return rate
}
func GetResolvers(resolvers string) []string {
	// handle resolver
	var rs []string
	var err error
	if resolvers != "" {
		rs, err = core.LinesInFile(resolvers)
		if err != nil {
			gologger.Fatalf("读取resolvers文件失败:%s\n", err.Error())
		}
		if len(rs) == 0 {
			gologger.Fatalf("resolvers文件内容为空\n")
		}
	} else {
		defaultDns := []string{
			"223.5.5.5",
			"223.6.6.6",
			"180.76.76.76",
			"119.29.29.29",
			"182.254.116.116",
			"114.114.114.115",
		}
		rs = defaultDns
	}
	return rs
}
func (opt *Options) Check() {

	if opt.Silent {
		gologger.MaxLevel = gologger.Silent
	}

	core.ShowBanner()

	if opt.Method == "verify" {
		if opt.Stdin {

		} else {
			if opt.FileName == "" || !core.FileExists(opt.FileName) {
				gologger.Fatalf("域名验证文件:%s 不存在! \n", opt.FileName)
			}
		}
	} else if opt.Method == "enum" {
		if opt.FileName != "" && !core.FileExists(opt.FileName) {
			gologger.Fatalf("字典文件:%s 不存在! \n", opt.FileName)
		}
		if opt.Stdin {

		} else {
			if len(opt.Domain) == 0 {
				gologger.Fatalf("域名未指定目标")
			}
		}
	}
}
func HasStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return false
	}
	return true
}
