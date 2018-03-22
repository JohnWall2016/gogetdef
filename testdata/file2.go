package testdata

import (
	"fmt"
	"strings"
)

func func1() {
	var dddd, aaaa string
	aaaa = "abc"
	dddd = aaaa
	fmt.Printf("%s\n", aaaa)
	fmt.Sprintf("%s\n", strings.Title(aaaa))
}
