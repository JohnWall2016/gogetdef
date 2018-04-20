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

func (ti *typeInfo) typeNode(obj types.Object) (path []ast.Node, node ast.Node) {
	if file := ti.fset.File(obj.Pos()); file != nil {
		path = ti.importer.PathEnclosingInterval(file.Name(), obj.Pos(), obj.Pos())
		for _, node = range path {
			switch node.(type) {
			case *ast.Ident:
				// continue ascending AST (searching for parent node of the identifier))
				continue
			default:
				return
			}
		}
	}
	return nil, nil
}

type funcsByName []*types.Func

func (p funcsByName) Len() int           { return len(p) }
func (p funcsByName) Less(i, j int) bool { return p[i].Name() < p[j].Name() }
func (p funcsByName) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (ti *typeInfo) ident(obj types.Object) (dcl *declaration, err error) {
	dcl = &declaration{}

	dcl.name = obj.Name()
	dcl.typ = obj.String()

	objPos := func(obj types.Object) string {
		if p := ti.fset.Position(obj.Pos()); p.IsValid() {
			return p.String()
		}
		return ""
	}

	dcl.pos = objPos(obj)

	nodes, node := ti.typeNode(obj)
	if node != nil {
		dcl.typ = formatNode(node, obj, ti.fset, *showall)
		if *showall {
			if s, ok := obj.Type().(*types.Named); ok && s.NumMethods() > 0 {
				var funcs funcsByName
				for i := 0; i < s.NumMethods(); i++ {
					funcs = append(funcs, s.Method(i))
				}
				sort.Sort(funcs)
				for _, m := range funcs {
					_, mnode := ti.typeNode(m)
					if mnode != nil {
						mtyp := formatNode(mnode, m, ti.fset, *showall)
						mpos := objPos(m)
						dcl.mthds = append(dcl.mthds, &typeAndPos{mtyp, mpos})
					}
				}
			}

			if nodes != nil {
				if obj.Pkg() != nil {
					dcl.imprt = obj.Pkg().Path()
				}
				for _, node := range nodes {
					//fmt.Printf("for %s: found %T\n%#v\n", id.Name, node, node)
					switch n := node.(type) {
					case *ast.Ident:
						continue
					case *ast.FuncDecl:
						dcl.doc = n.Doc.Text()
						return
					case *ast.Field:
						if n.Doc != nil {
							dcl.doc = n.Doc.Text()
						} else if n.Comment != nil {
							dcl.doc = n.Comment.Text()
						}
						return
					case *ast.TypeSpec:
						if n.Doc != nil {
							dcl.doc = n.Doc.Text()
							return
						}
						if n.Comment != nil {
							dcl.doc = n.Comment.Text()
							return
						}
					case *ast.ValueSpec:
						if n.Doc != nil {
							dcl.doc = n.Doc.Text()
							return
						}
						if n.Comment != nil {
							dcl.doc = n.Comment.Text()
							return
						}
					case *ast.GenDecl:
						constValue := ""
						if c, ok := obj.(*types.Const); ok {
							constValue = c.Val().ExactString()
						}
						if dcl.doc == "" && n.Doc != nil {
							dcl.doc = n.Doc.Text()
						}
						if constValue != "" {
							dcl.doc += fmt.Sprintf("\nConstant Value: %s", constValue)
						}
						return
					default:
						return
					}
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

func (ti *typeInfo) importSpec(spec *ast.ImportSpec) (dcl *declaration, err error) {
	path, _ := strconv.Unquote(spec.Path.Value)
	bpkg, err := build.Import(path, "", build.ImportComment)
	if err != nil {
		return
	}
	dcl = &declaration{typeAndPos: typeAndPos{typ: "package " + bpkg.Name, pos: bpkg.Dir}}
	if *showall {
		astPkg, ok := ti.importer.GetCachedPackage(bpkg.Name)
		if ok {
			docPkg := doc.New(astPkg, path, 0)
			dcl.doc = docPkg.Doc
			dcl.imprt = path
		}
	}
	return
}

func (ti *typeInfo) findDeclaration(fileName string, offset int) (dcl *declaration, err error) {
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
	cerr := types.NewChecker(conf, ti.fset, tpkg, &ti.Info, types.NoCheckUsage).Files(chkFiles)

	path, _ := imports.PathEnclosingInterval(astFile, pos, pos)

	for i, node := range path {
		switch n := node.(type) {
		case *ast.Ident:
			var obj types.Object
			if obj = ti.ObjectOf(n); obj == nil {
				continue
			}
			if v, ok := obj.(*types.Var); ok {
				if v.IsField() && v.Anonymous() { // embeded type field
					if i+1 < len(path) {
						p := path[i+1]
						i += 1
						if _, ok := p.(*ast.SelectorExpr); ok { // with a selector
							if i+1 < len(path) {
								p = path[i+1]
								i += 1
							}
						}
						if _, ok := p.(*ast.StarExpr); ok { // pointer type
							if i+1 < len(path) {
								p = path[i+1]
							}
						}
						// if in a struct's declaration, find the embeded field's own type
						if _, ok := p.(*ast.Field); ok {
							switch tn := v.Type().(type) {
							case *types.Named:
								obj = tn.Obj()
							case *types.Pointer:
								if tn, ok := tn.Elem().(*types.Named); ok {
									obj = tn.Obj()
								}
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
		cerr = errors.New("can't find the declaration")
	}
	return nil, cerr
}

func findDeclaration(fileName string, offset int, archive io.Reader) (dcl *declaration, err error) {
	var overlay map[string][]byte
	if archive != nil {
		overlay, err = imports.ParseOverlayArchive(archive)
		if err != nil {
			return
		}
	}
	ti := newTypeInfo(overlay)

	return ti.findDeclaration(fileName, offset)
}
