package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/parser"
	"go/types"
	"go/importer"
	"golang.org/x/tools/go/ast/astutil"
	"errors"
)

func findTypeInFile(filename string, src []byte, offset int) (decl, pos string, err error) {
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, filename, src, parser.AllErrors)
	if err != nil {
		return
	}

	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Scopes:     make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	_, err = conf.Check(astFile.Name.Name, fset, []*ast.File{astFile}, info)
	
	tokFile := fset.File(astFile.Pos())
	if tokFile == nil {
		return "", "", errors.New("can't get token file")
	}
	p := tokFile.Pos(offset)
	path, match := astutil.PathEnclosingInterval(astFile, p, p)

	if match {
		switch id := path[0].(type) {
		case *ast.Ident:
			fmt.Printf("%#v\n", info.ObjectOf(id))
		}
	}
	return
}
