package main

import (
	"errors"
	"fmt"
	"github.com/JohnWall2016/gogetdef/importer"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/ast/astutil"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var _ = fmt.Printf
var _ = build.Import

type typeInfo struct {
	types.Info
	fset     *token.FileSet
	importer *importer.Importer
	ctxt     *build.Context
	conf     *types.Config
	files    map[string]*ast.File
}

func newTypeInfo(overlay map[string][]byte) *typeInfo {
	info := &typeInfo{
		Info: types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Scopes:     make(map[ast.Node]*types.Scope),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
		},
		fset: token.NewFileSet(),
		ctxt: importer.OverlayContext(&build.Default, overlay),
		files: make(map[string]*ast.File),
	}
	info.importer = importer.New(info.ctxt, info.fset, info.files)
	info.conf = &types.Config{Importer: info.importer}
	return info
}

func (ti *typeInfo) ident(id *ast.Ident) (decl, pos string, err error) {
	obj := ti.ObjectOf(id)
	decl = obj.String()

	if p := ti.fset.Position(obj.Pos()); p.IsValid() {
		pos = p.String()
	}

	if file := ti.fset.File(obj.Pos()); file != nil {
		if astfile, ok := ti.files[file.Name()]; ok {
			nodes, _ := astutil.PathEnclosingInterval(astfile, obj.Pos(), obj.Pos())
			for _, node := range nodes {
				switch node.(type) {
				case *ast.Ident:
					// continue ascending AST (searching for parent node of the identifier))
					continue
				case *ast.FuncDecl, *ast.GenDecl, *ast.Field, *ast.TypeSpec, *ast.ValueSpec:
					// found the parent node
				default:
					break
				}
				decl = formatNode(node, obj, ti.fset, false)
				break
			}
		}
	}
	return
}

func (ti *typeInfo) importSpec(spec *ast.ImportSpec) (decl, pos string, err error) {
	path, _ := strconv.Unquote(spec.Path.Value)
	bpkg, err := build.Import(path, "", build.ImportComment)
	if err != nil {
		return
	}

	return "package " + bpkg.Name, bpkg.Dir, nil
}

func (ti *typeInfo) readDir(path string) ([]os.FileInfo, error) {
	if f := ti.ctxt.ReadDir; f != nil {
		return f(path)
	}
	return ioutil.ReadDir(path)
}

func (ti *typeInfo) openFile(path string) ([]byte, error) {
	if f := ti.ctxt.OpenFile; f != nil {
		file, err := f(path)
		if err == nil {
			defer file.Close()
			buf, err := ioutil.ReadAll(file)
			if err == nil {
				return buf, nil
			}
		}
	}
	return ioutil.ReadFile(path)
}

func (ti *typeInfo) parseDir(dir string, mode parser.Mode) (pkgs map[string]*ast.Package, first error) {
	list, err := ti.readDir(dir)
	if err != nil {
		return nil, err
	}

	pkgs = make(map[string]*ast.Package)
	for _, d := range list {
		if strings.HasSuffix(d.Name(), ".go") {
			filename := filepath.Join(dir, d.Name())

			var err error
			src, ok :=  ti.files[filename]
			if !ok {
				buf, err := ti.openFile(filename)
				if err != nil {
					buf = nil
				}
				src, err = parser.ParseFile(ti.fset, filename, buf, mode)
				if err == nil {
					ti.files[filename] = src
				}
			}
			
			if err == nil {
				name := src.Name.Name
				pkg, found := pkgs[name]
				if !found {
					pkg = &ast.Package{
						Name:  name,
						Files: make(map[string]*ast.File),
					}
					pkgs[name] = pkg
				}
				pkg.Files[filename] = src
			} else if first == nil {
				first = err
			}
		}
	}

	return
}

func sameFile(a, b string) bool {
	if filepath.Base(a) != filepath.Base(b) {
		// We only care about symlinks for the GOPATH itself. File
		// names need to match.
		return false
	}
	if ai, err := os.Stat(a); err == nil {
		if bi, err := os.Stat(b); err == nil {
			return os.SameFile(ai, bi)
		}
	}
	return false
}

func (ti *typeInfo) findDeclare(filename string, offset int) (decl, pos string, err error) {
	pkgs, err := ti.parseDir(filepath.Dir(filename), parser.ParseComments)
	if err != nil {
		return
	}

	var pkgName string
	var astFile *ast.File
	var astFiles []*ast.File
	for pname, pkg := range pkgs {
		astFiles = make([]*ast.File, 0, len(pkg.Files))
		for fname, afile := range pkg.Files {
			astFiles = append(astFiles, afile)
			if sameFile(filename, fname) {
				pkgName = pname
				astFile = afile
			}
		}
	}

	if pkgName == "" {
		return "", "", errors.New("can't get package name")
	}

	_, cerr := ti.conf.Check(pkgName, ti.fset, astFiles, &ti.Info)

	tokFile := ti.fset.File(astFile.Pos())
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
			if obj := ti.ObjectOf(n); obj == nil {
				continue
			}
			return ti.ident(n)
		case *ast.ImportSpec:
			return ti.importSpec(n)
		}
	}
	return "", "", cerr
}

func findTypeSpec(decl *ast.GenDecl, pos token.Pos) *ast.TypeSpec {
	for _, spec := range decl.Specs {
		typeSpec := spec.(*ast.TypeSpec)
		if typeSpec.Pos() == pos {
			return typeSpec
		}
	}
	return nil
}

func findVarSpec(decl *ast.GenDecl, pos token.Pos) *ast.ValueSpec {
	for _, spec := range decl.Specs {
		varSpec := spec.(*ast.ValueSpec)
		for _, ident := range varSpec.Names {
			if ident.Pos() == pos {
				return varSpec
			}
		}
	}
	return nil
}

func FindDeclare(filename string, offset int, archive io.Reader) (decl, pos string, err error) {
	var overlay map[string][]byte
	if archive != nil {
		overlay, err = importer.ParseOverlayArchive(archive)
		if err != nil {
			return
		}
	}
	ti := newTypeInfo(overlay)

	return ti.findDeclare(filename, offset)
}
