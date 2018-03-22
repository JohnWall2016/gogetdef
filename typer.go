package main

import (
	"errors"
	"fmt"
	"github.com/JohnWall2016/gogetdef/srcimporter"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/ast/astutil"
)

var _ = fmt.Printf
var _ = build.Import
var _ = importer.Default

type typeInfo struct {
	types.Info
	fset       *token.FileSet
	conf       *types.Config
	importPkgs map[string]*types.Package
}

func newTypeInfo() *typeInfo {
	info := &typeInfo{
		Info: types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Scopes:     make(map[ast.Node]*types.Scope),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
		},
		fset:       token.NewFileSet(),
		importPkgs: make(map[string]*types.Package),
	}
	info.conf = &types.Config{Importer: srcimporter.New(&build.Default, info.fset, info.importPkgs)}
	return info
}

func (ti *typeInfo) ident(id *ast.Ident) (decl, pos string, err error) {
	obj := ti.ObjectOf(id)
	return obj.String(), ti.fset.Position(obj.Pos()).String(), nil
}

func findTypeInFile(filename string, src []byte, offset int) (decl, pos string, err error) {
	info := newTypeInfo()

	astFile, err := parser.ParseFile(info.fset, filename, src, parser.ParseComments)
	if err != nil {
		return
	}

	_, cerr := info.conf.Check(astFile.Name.Name, info.fset, []*ast.File{astFile}, &info.Info)

	tokFile := info.fset.File(astFile.Pos())
	if tokFile == nil {
		return "", "", errors.New("can't get token file")
	}
	if offset > tokFile.Size() {
		return "", "", errors.New("illegal file offset")
	}
	p := tokFile.Pos(offset)
	path, _ := astutil.PathEnclosingInterval(astFile, p, p)

	for _, node := range path {
		switch n := node.(type) {
		case *ast.Ident:
			if obj := info.ObjectOf(n); obj == nil {
				continue
			}
			return info.ident(n)
		default:
			panicNode(n)
		}
	}
	return "", "", cerr
}
