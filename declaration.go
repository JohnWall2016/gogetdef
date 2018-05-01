package main

import (
	"bytes"
	"fmt"
	"go/doc"
)

type typePos struct {
	typ, pos string
}

type declaration struct {
	name string
	typePos
	imprt string
	doc   string
	mthds []*typePos
}

func (d *declaration) String() string {
	buf := &bytes.Buffer{}
	if d.imprt != "" {
		fmt.Fprintf(buf, "import \"%s\"\n\n", d.imprt)
	}
	fmt.Fprintf(buf, "%s\n\n", d.typ)
	if d.doc != "" {
		doc.ToText(buf, d.doc, "", "    ", 80)
		fmt.Fprintln(buf)
	}
	if len(d.mthds) > 0 {
		fmt.Fprintf(buf, "[:method:")
		for _, m := range d.mthds {
			fmt.Fprintf(buf, "[%s|%s]", m.typ, m.pos)
		}
		fmt.Fprintf(buf, "]")
	}
	return buf.String()
}
