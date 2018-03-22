package main

import (
	"fmt"
	"runtime"
)

func main() {
	pc, file, line, ok := runtime.Caller(0)
	fmt.Println(pc, file, line, ok)
	//parseSingleFile(ioutil.ReadFile(
}
