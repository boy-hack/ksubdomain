package runner

import (
	"context"
)

func (r *runner) handleResult(ctx context.Context) {

	//onlyDomain := r.options.OnlyDomain
	//notPrint := r.options.NotPrint
	for result := range r.recver {
		for _, out := range r.options.Writer {
			_ = out.WriteDomainResult(result)
		}

		//if onlyDomain {
		//	msg = result.Subdomain
		//} else {
		//	var content = []string{
		//		result.Subdomain,
		//	}
		//	content = append(content, result.Answers...)
		//	msg = strings.Join(content, " => ")
		//}

		//if !notPrint {
		//	if !r.options.Silent {
		//		// 打印一下结果,可以看得更直观
		//		r.PrintStatus()
		//	} else {
		//		gologger.Silentf("%s\n", msg)
		//	}
		//}
	}
}
