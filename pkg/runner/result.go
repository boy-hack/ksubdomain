package runner

func (r *Runner) handleResult() {
	for result := range r.recver {
		for _, out := range r.options.Writer {
			_ = out.WriteDomainResult(result)
		}
		r.printStatus()
	}
}
