package processbar

import "fmt"

type ScreenProcess struct {
	Silent bool
}

func (s *ScreenProcess) WriteData(data *ProcessData) {
	if !s.Silent {
		fmt.Printf("\rSuccess:%d Send:%d Queue:%d Accept:%d Fail:%d Elapsed:%ds", data.SuccessIndex, data.SendIndex, data.QueueLength, data.RecvIndex, data.FaildIndex, data.Elapsed)
	}
}

func (s *ScreenProcess) Close() {
}
