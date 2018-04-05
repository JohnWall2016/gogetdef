package main

import (
	"errors"
	"fmt"
	"github.com/JohnWall2016/gogetdef/imports"
	"github.com/JohnWall2016/gogetdef/parser"
	"github.com/JohnWall2016/gogetdef/types"
	"go/ast"
	"go/build"
	"go/doc"
	"go/token"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var _ = fmt.Printf

type typeInfo struct {
	types.Info
	fset     *token.FileSet
	importer *imports.Importer
	ctxt     *build.Context
	errors   []error
	maxerrs  int
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
		fset:    token.NewFileSet(),
		ctxt:    imports.OverlayContext(&build.Default, overlay),
		maxerrs: 10,
	}
	var mode parser.Mode
	if *showall {
		mode = parser.ParseComments
	}
	info.importer = imports.NewImporter(info.ctxt, info.fset, &info.Info, mode)

	return info
}

func (ti *typeInfo) ident(obj types.Object) (def *definition, err error) {
	def = &definition{}

	def.decl = obj.String()
	if p := ti.fset.Position(obj.Pos()); p.IsValid() {
		def.pos = p.String()
	}

	if file := ti.fset.File(obj.Pos()); file != nil {
		nodes := ti.importer.PathEnclosingInterval(file.Name(), obj.Pos(), obj.Pos())
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
			def.decl = formatNode(node, obj, ti.fset, *showall)
			break
		}
		if *showall {
			if obj.Pkg() != nil {
				def.imprt = obj.Pkg().Path()
			}
			for _, node := range nodes {
				//fmt.Printf("for %s: found %T\n%#v\n", id.Name, node, node)
				switch n := node.(type) {
				case *ast.Ident:
					continue
				case *ast.FuncDecl:
					def.doc = n.Doc.Text()
					return
				case *ast.Field:
					if n.Doc != nil {
						def.doc = n.Doc.Text()
					} else if n.Comment != nil {
						def.doc = n.Comment.Text()
					}
					return
				case *ast.TypeSpec:
					if n.Doc != nil {
						def.doc = n.Doc.Text()
						return
					}
					if n.Comment != nil {
						def.doc = n.Comment.Text()
						return
					}
				case *ast.ValueSpec:
					if n.Doc != nil {
						def.doc = n.Doc.Text()
						return
					}
					if n.Comment != nil {
						def.doc = n.Comment.Text()
						return
					}
				case *ast.GenDecl:
					constValue := ""
					if c, ok := obj.(*types.Const); ok {
						constValue = c.Val().ExactString()
					}
					if def.doc == "" && n.Doc != nil {
						def.doc = n.Doc.Text()
					}
					if constValue != "" {
						def.doc += fmt.Sprintf("\nConstant Value: %s", constValue)
					}
					return
				default:
					return
				}
			}
		}
	} else if obj.Pkg() == nil {
		bt, err := ti.importer.Import("builtin")
		if err == nil {
			obj := bt.Scope().Lookup(obj.Name())
			if obj != nil {
				return ti.ident(obj)
			}
		}
	}
	return
}

func (ti *typeInfo) importSpec(spec *ast.ImportSpec) (def *definition, err error) {
	path, _ := strconv.Unquote(spec.Path.Value)
	bpkg, err := build.Import(path, "", build.ImportComment)
	if err != nil {
		return
	}
	def = &definition{decl: "package " + bpkg.Name, pos: bpkg.Dir}
	if *showall {
		astPkg, ok := ti.importer.GetCachedPackage(bpkg.Name)
		if ok {
			docPkg := doc.New(astPkg, path, 0)
			def.doc = docPkg.Doc
			def.imprt = path
		}
	}
	return
}

func (ti *typeInfo) findDefinition(fileName string, offset int) (def *definition, err error) {
	astFile, err := ti.importer.ParseFile(fileName,
		func(lbrace, rbrace int) bool {
			if lbrace <= offset && offset <= rbrace {
				return true
			}
			return false
		})
	if err != nil {
		return
	}

	pkgName := astFile.Name.Name
	if pkgName == "" {
		err = errors.New("can't get package name")
		return
	}

	tokFile := ti.fset.File(astFile.Pos())
	if tokFile == nil {
		return nil, errors.New("can't get token file")
	}
	if offset > tokFile.Size() {
		return nil, errors.New("illegal file offset")
	}
	pos := tokFile.Pos(offset)

	astFiles, err := ti.importer.ParseDir(filepath.Dir(fileName))
	if err != nil {
		return
	}

	chkFiles := []*ast.File{astFile}
	for _, afile := range astFiles {
		if afile.Name.Name == pkgName && afile != astFile {
			chkFiles = append(chkFiles, afile)
		}
	}

	if strings.HasSuffix(fileName, "_test.go") {
		rpkg := strings.TrimSuffix(pkgName, "_test")
		ti.importer.IncludeTests = func(pkg string) bool {
			if pkg == rpkg {
				return true
			}
			return false
		}
	} else {
		ti.importer.IncludeTests = nil
	}

	conf := &types.Config{
		Importer: ti.importer,
		CheckFuncBodies: func(lbrace, rbrace token.Pos) bool {
			if lbrace <= pos && pos <= rbrace {
				return true
			}
			return false
		},
		FakeImportC: true,
		Error: func(err error) {
			if len(ti.errors) <= ti.maxerrs+1 {
				ti.errors = append(ti.errors, err)
			}
		},
	}
	tpkg := types.NewPackage(pkgName, "")
	cerr := types.NewChecker(conf, ti.fset, tpkg, &ti.Info).Files(chkFiles)

	path, _ := imports.PathEnclosingInterval(astFile, pos, pos)

	for i, node := range path {
		switch n := node.(type) {
		case *ast.Ident:
			var obj types.Object
			if obj = ti.ObjectOf(n); obj == nil {
				continue
			}
			if v, ok := obj.(*types.Var); ok {
				if v.IsField() && v.Anonymous() {
					if i + 1 < len(path) {
						p := path[i+1]
						if _, ok := p.(*ast.SelectorExpr); ok {
							if i + 2 < len(path) {
								p = path[i+2]
							}
						}
						if _, ok := p.(*ast.Field); ok {
							if tn, ok := v.Type().(*types.Named); ok {
								obj = tn.Obj()
							}
						}
					}
				}
			}
			return ti.ident(obj)
		case *ast.ImportSpec:
			return ti.importSpec(n)
		default:
			break
		}
	}
	if cerr != nil && *showall {
		errmsg := []string{}
		for _, e := range ti.errors {
			errmsg = append(errmsg, e.Error())
		}
		sort.Strings(errmsg)
		if len(errmsg) > ti.maxerrs {
			errmsg[ti.maxerrs+1] = "..."
		}
		cerr = errors.New(strings.Join(errmsg, "\n"))
	}
	if cerr == nil {
		//cerr = errors.New(fmt.Sprintf("can't found the node: %#v", node))
		cerr = errors.New("can't find definition")
	}
	return nil, cerr
}

func findDefinition(fileName string, offset int, archive io.Reader) (def *definition, err error) {
	var overlay map[string][]byte
	if archive != nil {
		overlay, err = imports.ParseOverlayArchive(archive)
		if err != nil {
			return
		}
	}
	ti := newTypeInfo(overlay)

	return ti.findDefinition(fileName, offset)
}
