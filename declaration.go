package main

import (
	"bytes"
	"fmt"
	"go/doc"
)

type declaration struct {
	typ, pos   string
	imprt, doc string
	mthds      []string
}

func (d *declaration) String() string {
	buf := &bytes.Buffer{}
	if d.imprt != "" {
		fmt.Fprintf(buf, "import \"%s\"\n\n", d.imprt)
	}
	fmt.Fprintf(buf, "%s\n\n", d.typ)
	/*if d.doc == "" {
		d.doc = "Undocumented."
	}*/
	if d.doc != "" {
		doc.ToText(buf, d.doc, "", "    ", 80)
		fmt.Fprintln(buf)
	}

	if len(d.mthds) > 0 {
		for _, m := range d.mthds {
			fmt.Fprintf(buf, "%s\n", m)
		}
		fmt.Fprintln(buf)
	}

	if d.pos != "" {
		fmt.Fprintf(buf, "[%s]\n", d.pos)
	}
	return buf.String()
}
