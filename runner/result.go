package runner

import (
	"context"
)

func (r *runner) handleResult(ctx context.Context) {
	for {
		select {
		case result := <-r.recver:
			for _, out := range r.options.Writer {
				_ = out.WriteDomainResult(result)
			}
			r.printStatus()

		case <-ctx.Done():
			return
		}
	}
}
