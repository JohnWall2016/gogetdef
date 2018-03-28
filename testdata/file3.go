package testdata

import (
	"fmt"
)

func puts(msg ...interface{}) {
	msg = append(msg, "abc")
	fmt.Println(msg)
}
