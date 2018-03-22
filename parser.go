package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/tools/go/ast/astutil"
	"strings"
)

func panicNode(node interface{}) {
	panic(fmt.Sprintf("not processed node: %#v", node))
}

func readFuncType(ft *ast.FuncType) string {
	pars := []string{}
	if ft.Params != nil {
		for _, field := range ft.Params.List {
			finfo := readField(field, "")
			pars = append(pars, finfo)
		}
	}
	ress := []string{}
	if ft.Results != nil {
		for _, field := range ft.Results.List {
			rinfo := readField(field, "")
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

func readField(field *ast.Field, name string) string {
	if name == "" {
		names := []string{}
		for _, nm := range field.Names {
			names = append(names, nm.Name)
		}
		name = strings.Join(names, ", ")
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
			fields = append(fields, readField(field, ""))
		}
	}
	if len(fields) == 0 {
		return "interface{}"
	} else {
		return fmt.Sprintf("interface {\n\t%s\n}", strings.Join(fields, "\n\t"))
	}
}

func readUnaryExpr(expr *ast.UnaryExpr) string {
	return fmt.Sprintf("%s %s\n", expr.Op, readExpr(expr.X))
}

func readExpr(node ast.Expr) string {
	switch expr := node.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.BasicLit:
		return expr.Value
	case *ast.FuncLit:
		return "func" + readFuncType(expr.Type)
	case *ast.UnaryExpr:
		return readUnaryExpr(expr)
	default:
		panicNode(expr)
	}
	return ""
}

func expandAssignStmt(name string, node *ast.AssignStmt, kind ast.ObjKind) (decl string, pos token.Pos) {
	lhsIndx := 0
	for i, expr := range node.Lhs {
		if id, ok := expr.(*ast.Ident); ok && id.Name == name {
			lhsIndx = i
			pos = id.Pos()
		}
	}

	values := []string{}
	if len(node.Lhs) > 0 && len(node.Rhs) == len(node.Lhs) {
		values = append(values, readExpr(node.Rhs[lhsIndx]))
	} else {
		for _, expr := range node.Rhs {
			values = append(values, readExpr(expr))
		}
	}
	return fmt.Sprintf("%s %s = %s", kind, name, strings.Join(values, ", ")), pos
}

func expandValueSpec(name string, v *ast.ValueSpec, kind ast.ObjKind) (decl string, pos token.Pos) {
	for _, nm := range v.Names {
		if nm.Name == name {
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

			decl = fmt.Sprintf("%s %s", kind, name)
			if typ != "" {
				decl = fmt.Sprintf("%s %s", decl, typ)
			}
			if len(values) > 0 {
				decl = fmt.Sprintf("%s = %s", decl, strings.Join(values, ", "))
			}
			pos = nm.Pos()
		}
	}

	return
}

func expandField(name string, field *ast.Field, kind ast.ObjKind) (decl string, pos token.Pos) {
	//fmt.Printf("%s, %s, %#v", name, kind, field)

	for _, nm := range field.Names {
		if nm.Name == name {
			fld := readField(field, name)
			decl = fmt.Sprintf("%s %s", kind, fld)
			pos = nm.Pos()
		}
	}
	return
}

func expandDeclare(obj *ast.Object) (decl string, pos token.Pos) {
	pos = obj.Decl.(ast.Node).Pos()

	switch d := obj.Decl.(type) {
	case *ast.AssignStmt:
		return expandAssignStmt(obj.Name, d, obj.Kind)
	case *ast.ValueSpec:
		return expandValueSpec(obj.Name, d, obj.Kind)
	case *ast.Field:
		return expandField(obj.Name, d, obj.Kind)
	default:
		panicNode(obj.Decl)
	}
	return
}

func expandNode(node ast.Node) (decl string, pos token.Pos) {
	switch n := node.(type) {
	case *ast.Ident:
		if n.Obj != nil {
			if n.Obj.Kind == ast.Var {
				if n.Obj.Decl != nil {
					decl, pos = expandDeclare(n.Obj)
					break
				}
			}
			panicNode(n.Obj)
		}
		panicNode(n)
	case *ast.CallExpr:
		return expandNode(n.Fun)
	default:
		panicNode(n)
	}
	return
}

func findDeclareInFile(filename string, src []byte, offset int) (decl, pos string, err error) {
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, filename, src, parser.AllErrors)
	if err != nil {
		return
	}
	tokFile := fset.File(astFile.Pos())
	if tokFile == nil {
		return "", "", errors.New("can't get token file")
	}
	p := tokFile.Pos(offset)
	path, match := astutil.PathEnclosingInterval(astFile, p, p)

	if match {
		dpos := token.NoPos
		decl, dpos = expandNode(path[0])

		if dpos == token.NoPos {
			pos = ""
		} else {
			pos = tokFile.Position(dpos).String()
		}
	}
	return decl, pos, nil
}
