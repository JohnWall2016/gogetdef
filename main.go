package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var (
	pos      = flag.String("pos", "", "Filename and byte offset of item to document, e.g. foo.go:#123")
	modified = flag.Bool("modified", false, "read an archive of modified files from standard input")
)

const modifiedUsage = `
The archive format for the -modified flag consists of the file name, followed
by a newline, the decimal file size, another newline, and the contents of the file.

This allows editors to supply gogetdef with the contents of their unsaved buffers.
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, modifiedUsage)
	}
	flag.Parse()

	filename, offset, err := parsePos(*pos)
	//fmt.Printf("%s:%d\n", filename, offset)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var archive io.Reader
	if *modified {
		archive = os.Stdin
	}

	decl, pos, err := FindDeclare(filename, int(offset), archive)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("gogetdef-return")
	fmt.Println(decl)
	fmt.Println(pos)
}

func parsePos(p string) (filename string, offset int64, err error) {
	// foo.go:#123
	if p == "" {
		err = errors.New("missing required -pos flag")
		return
	}
	sep := strings.LastIndex(p, ":")
	// need at least 2 characters after the ':'
	// (the # sign and the offset)
	if sep == -1 || sep > len(p)-2 || p[sep+1] != '#' {
		err = fmt.Errorf("invalid option: -pos=%s", p)
		return
	}
	filename = p[:sep]
	offset, err = strconv.ParseInt(p[sep+2:], 10, 32)
	return
}
