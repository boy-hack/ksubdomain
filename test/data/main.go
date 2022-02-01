package main

import (
	"bufio"
	"fmt"
	"ksubdomain/core/gologger"
	"os"
)

func main() {
	f2, err := os.Open("./test/data/verify.txt")
	if err != nil {
		gologger.Fatalf("打开文件:%s 出现错误:%s", "verify.txt", err.Error())
	}
	defer f2.Close()
	reader := bufio.NewReader(f2)
	for {
		n, _, err := reader.ReadLine()
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Println(string(n))

	}
}
