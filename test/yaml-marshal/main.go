package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"net"
)

type testYaml struct {
	A string  `yaml:"a"`
	B SelfMac `yaml:"b"`
}

type SelfMac net.HardwareAddr

func (d SelfMac) MarshalYAML() (interface{}, error) {
	n := (net.HardwareAddr)(d)
	return n.String(), nil
}

func (d *SelfMac) UnmarshalYAML(value *yaml.Node) error {
	v := value.Value
	v2, err := net.ParseMAC(v)
	if err != nil {
		return err
	}
	n := SelfMac(v2)
	*d = n
	return nil
}

func main() {
	b, err := net.ParseMAC("00:00:5e:00:53:01")
	if err != nil {
		fmt.Println(err)
		return
	}
	b2 := SelfMac(b)
	a := testYaml{
		A: "testa",
		B: b2,
	}
	data, err := yaml.Marshal(a)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))

	var out testYaml
	fmt.Println("反序列化后...")
	err = yaml.Unmarshal(data, &out)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(out)

}
