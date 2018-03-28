package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"path/filepath"
	"strings"
)

func (ti *typeInfo) builtinFile(fs *token.FileSet) (*ast.File, error) {
	bpkg, err := build.Import("builtin", "", build.FindOnly)
	// should never fail
	if err != nil {
		panic(err)
	}
	return ti.importer.ParseFile(filepath.Join(bpkg.Dir, "builtin.go"))
}

func readFuncType(ft *ast.FuncType) string {
	pars := []string{}
	if ft.Params != nil {
		for _, field := range ft.Params.List {
			finfo := readField(field)
			pars = append(pars, finfo)
		}
	}
	ress := []string{}
	if ft.Results != nil {
		for _, field := range ft.Results.List {
			rinfo := readField(field)
			ress = append(ress, rinfo)
		}
	}
	info := fmt.Sprintf("(%s)", strings.Join(pars, ", "))
	if len(ress) == 1 {
		info += " " + ress[0]
	} else if len(ress) > 1 {
		info += fmt.Sprintf(" (%s)", strings.Join(ress, ", "))
	}

	return info
}

func readField(field *ast.Field) string {
	name := ""
	for _, nm := range field.Names {
		name = nm.Name
	}
	typ := ""
	if field.Type != nil {
		switch ft := field.Type.(type) {
		case *ast.FuncType:
			typ = readFuncType(ft)
			return name + typ
		case *ast.Ident:
			typ = ft.Name
		case *ast.ArrayType:
			if ft.Elt != nil {
				if id, ok := ft.Elt.(*ast.Ident); ok {
					typ = "[]" + id.String()
				}
			}
		case *ast.Ellipsis:
			if ft.Elt != nil {
				if id, ok := ft.Elt.(*ast.Ident); ok {
					typ = "..." + id.String()
				}
			}
		case *ast.MapType:
			key := ""
			if ft.Key != nil {
				if id, ok := ft.Key.(*ast.Ident); ok {
					key = id.String()
				}
			}
			value := ""
			if ft.Value != nil {
				if id, ok := ft.Value.(*ast.Ident); ok {
					value = id.String()
				}
			}
			typ = fmt.Sprintf("map[%s]%s", key, value)
		case *ast.StarExpr:
			if ft.X != nil {
				if id, ok := ft.X.(*ast.Ident); ok {
					typ = "*" + id.Name
				}
			}
		case *ast.ChanType:
			if ft.Value != nil {
				if id, ok := ft.Value.(*ast.Ident); ok {
					val := id.Name
					if ft.Dir == ast.SEND {
						typ = "chan<- " + val
					} else if ft.Dir == ast.RECV {
						typ = "<-chan " + val
					} else {
						typ = "chan " + val
					}
				}
			}
		case *ast.InterfaceType:
			typ = readInterfaceType(ft)
		default:
			fmt.Printf("%#v\n", field)
		}
	}
	if name == "" {
		return typ
	} else if typ == "" {
		return name
	} else {
		return name + " " + typ
	}
}

func readInterfaceType(it *ast.InterfaceType) string {
	fields := []string{}
	if it.Methods != nil {
		for _, field := range it.Methods.List {
			fields = append(fields, readField(field))
		}
	}
	if len(fields) == 0 {
		return "interface{}"
	} else {
		return fmt.Sprintf("interface {\n\t%s\n}", strings.Join(fields, "\n\t"))
	}
}

func getValueSpec(name string, v *ast.ValueSpec, vt token.Token, vdoc *ast.CommentGroup, fs *token.FileSet) *definition {
	for _, nm := range v.Names {
		if name == nm.Name {
			typ := ""
			if v.Type != nil {
				if t, ok := v.Type.(*ast.Ident); ok {
					typ = t.Name
				}
			}

			values := []string{}
			for _, expr := range v.Values {
				if basicLit, ok := expr.(*ast.BasicLit); ok {
					values = append(values, basicLit.Value)
				}
				if binExpr, ok := expr.(*ast.BinaryExpr); ok {
					if x, ok := binExpr.X.(*ast.BasicLit); ok {
						if y, ok := binExpr.X.(*ast.BasicLit); ok {
							values = append(values, fmt.Sprintf("%s %s %s", x.Value, binExpr.Op, y.Value))
						}
					}
				}
			}

			def := &definition{}

			decl := fmt.Sprintf("%s %s", vt, name)
			if typ != "" {
				decl = fmt.Sprintf("%s %s", decl, typ)
			}
			if len(values) > 0 {
				decl = fmt.Sprintf("%s = %s", decl, strings.Join(values, ", "))
			}
			def.decl = decl

			pos := nm.Pos()
			if pos.IsValid() {
				def.pos = fs.Position(pos).String()
			}

			sdoc := ""
			if v.Doc != nil {
				sdoc = v.Doc.Text()
			} else if vdoc != nil {
				sdoc = vdoc.Text()
			}
			def.doc = sdoc

			return def
		}
	}

	return nil
}

func getTypeSpec(name string, t *ast.TypeSpec, tt token.Token, tdoc *ast.CommentGroup, fs *token.FileSet) *definition {
	if name != t.Name.Name {
		return nil
	}

	def := &definition{}

	decl := fmt.Sprintf("%s %s", tt, name)
	typ := ""
	if t.Type != nil {
		if ti, ok := t.Type.(*ast.Ident); ok {
			typ = ti.Name
		} else if ti, ok := t.Type.(*ast.InterfaceType); ok {
			typ += readInterfaceType(ti)
		}
	}
	if typ != "" {
		if t.Assign != 0 {
			decl = fmt.Sprintf("%s = %s", decl, typ)
		} else {
			decl = fmt.Sprintf("%s %s", decl, typ)
		}
	}
	def.decl = decl

	pos := t.Name.Pos()
	if pos.IsValid() {
		def.pos = fs.Position(pos).String()
	}

	sdoc := ""
	if t.Doc != nil {
		sdoc = t.Doc.Text()
	} else if tdoc != nil {
		sdoc = tdoc.Text()
	}
	def.doc = sdoc

	return def
}

func getFuncDecl(name string, f *ast.FuncDecl, fs *token.FileSet) *definition {
	if f.Name.Name != name {
		return nil
	}

	def := &definition{}

	if f.Doc != nil {
		def.doc = f.Doc.Text()
	}

	pos := f.Name.Pos()
	if pos.IsValid() {
		def.pos = fs.Position(pos).String()
	}

	decl := "func"
	recv := []string{}
	if f.Recv != nil {
		for _, field := range f.Recv.List {
			recv = append(recv, readField(field))
		}
	}
	if len(recv) > 0 {
		decl += " (" + strings.Join(recv, ", ") + ")"
	}
	decl += " " + name
	decl += readFuncType(f.Type)
	def.decl = decl

	return def
}

func (ti *typeInfo) findBuiltinDef(name string) (def *definition, err error) {
	f, err := ti.builtinFile(ti.fset)
	if err != nil {
		return
	}

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.CONST, token.VAR:
				for _, spec := range d.Specs {
					def = getValueSpec(name, spec.(*ast.ValueSpec), d.Tok, d.Doc, ti.fset)
					if def != nil {
						break
					}
				}
			case token.TYPE:
				for _, spec := range d.Specs {
					def = getTypeSpec(name, spec.(*ast.TypeSpec), d.Tok, d.Doc, ti.fset)
					if def != nil {
						break
					}
				}
			}
		case *ast.FuncDecl:
			def = getFuncDecl(name, d, ti.fset)
		}
		if def != nil {
			def.imprt = "builtin"
			return def, nil
		}
	}

	return nil, nil
}
