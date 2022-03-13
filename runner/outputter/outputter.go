package outputter

import (
	"os"
)

type FileOutPut struct {
	output *os.File
}

func NewFileOutput(filename string) (*FileOutPut, error) {
	output, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return nil, err
	}
	f := new(FileOutPut)
	f.output = output
	return f, err
}
func (f *FileOutPut) Write(b []byte) (n int, err error) {
	return f.output.Write(b)
}
func (f *FileOutPut) Close() error {
	return f.output.Close()
}
