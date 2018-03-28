package main

import (
	"bytes"
	"fmt"
	"go/doc"
)

type definition struct {
	decl, pos  string
	imprt, doc string
}

func (d *definition) String() string {
	buf := &bytes.Buffer{}
	if d.imprt != "" {
		fmt.Fprintf(buf, "import \"%s\"\n\n", d.imprt)
	}
	fmt.Fprintf(buf, "%s\n\n", d.decl)
	if d.doc == "" {
		d.doc = "Undocumented."
	}
	doc.ToText(buf, d.doc, "", "    ", 80)
	if d.pos != "" {
		fmt.Fprintf(buf, "\n[%s]\n", d.pos)
	}
	return buf.String()
}
