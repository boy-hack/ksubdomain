package main

import (
	"fmt"
	"github.com/boy-hack/hmap/store/hybrid"
)

func main() {
	hm, err := hybrid.New(hybrid.DefaultDiskOptions)
	if err != nil {
		fmt.Errorf(err.Error())
		return
	}
	hm.Set("test11", nil)
	hm.Set("test12", nil)
	hm.Set("test13", nil)
	hm.Set("test14", nil)
	hm.Set("test15", nil)
	fmt.Println(hm.Size())

	hm.Scan(func(bytes []byte, bytes2 []byte) error {
		fmt.Println(string(bytes), bytes2)
		return nil
	})
	hm.Del("test11")
	fmt.Println("---------------------")
	hm.Scan(func(bytes []byte, bytes2 []byte) error {
		fmt.Println(string(bytes), bytes2)
		return nil
	})
	fmt.Println(hm.Empty())

}
