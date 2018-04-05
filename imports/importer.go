// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package srcimporter implements importing directly
// from source files rather than installed packages.
package imports

import (
	"fmt"
	"github.com/JohnWall2016/gogetdef/parser"
	"github.com/JohnWall2016/gogetdef/types"
	"go/ast"
	"go/build"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// An Importer provides the context for importing packages from source code.
type Importer struct {
	ctxt         *build.Context
	fset         *token.FileSet
	sizes        types.Sizes
	typPkgs      map[string]*types.Package
	astPkgs      *astPkgCache
	info         *types.Info
	IncludeTests func(pkg string) bool
	mode         parser.Mode
}

type astPkgCache struct {
	sync.RWMutex
	packages map[string]*ast.Package
}

func (c *astPkgCache) cachedFile(name string) (*ast.File, bool) {
	c.RLock()
	defer c.RUnlock()
	for _, pkg := range c.packages {
		f, cached := pkg.Files[name]
		if cached {
			return f, cached
		}
	}
	return nil, false
}

func (c *astPkgCache) cacheFile(name string, file *ast.File) {
	c.Lock()
	defer c.Unlock()
	pkgName := file.Name.Name
	if pkg, ok := c.packages[pkgName]; ok {
		pkg.Files[name] = file
	} else {
		pkg = &ast.Package{
			Name: pkgName,
			Files: map[string]*ast.File{
				name: file,
			},
		}
		c.packages[pkgName] = pkg
	}
}

func (c *astPkgCache) cachedPackage(pkgName string) (pkg *ast.Package, ok bool) {
	c.RLock()
	defer c.RUnlock()
	pkg, ok = c.packages[pkgName]
	return
}

// NewImporter returns a new Importer for the given context, file set, and map
// of packages. The context is used to resolve import paths to package paths,
// and identifying the files belonging to the package. If the context provides
// non-nil file system functions, they are used instead of the regular package
// os functions. The file set is used to track position information of package
// files; and imported packages are added to the packages map.
func NewImporter(ctxt *build.Context, fset *token.FileSet, info *types.Info, mode parser.Mode) *Importer {
	return &Importer{
		ctxt:    ctxt,
		fset:    fset,
		sizes:   types.SizesFor(ctxt.Compiler, ctxt.GOARCH), // uses go/types default if GOARCH not found
		typPkgs: make(map[string]*types.Package),
		astPkgs: &astPkgCache{packages: make(map[string]*ast.Package)},
		info:    info,
		mode:    mode,
	}
}

// Importing is a sentinel taking the place in Importer.packages
// for a package that is in the process of being imported.
var importing types.Package

// Import(path) is a shortcut for ImportFrom(path, "", 0).
func (p *Importer) Import(path string) (*types.Package, error) {
	return p.ImportFrom(path, "", 0)
}

// ImportFrom imports the package with the given import path resolved from the given srcDir,
// adds the new package to the set of packages maintained by the importer, and returns the
// package. Package path resolution and file system operations are controlled by the context
// maintained with the importer. The import mode must be zero but is otherwise ignored.
// Packages that are not comprised entirely of pure Go files may fail to import because the
// type checker may not be able to determine all exported entities (e.g. due to cgo dependencies).
func (p *Importer) ImportFrom(path, srcDir string, mode types.ImportMode) (*types.Package, error) {
	if mode != 0 {
		panic("non-zero import mode")
	}
	// determine package path (do vendor resolution)
	var bp *build.Package
	var err error
	switch {
	default:
		if abs, err := p.absPath(srcDir); err == nil { // see issue #14282
			srcDir = abs
		}
		bp, err = p.ctxt.Import(path, srcDir, build.FindOnly)

	case build.IsLocalImport(path):
		// "./x" -> "srcDir/x"
		bp, err = p.ctxt.ImportDir(filepath.Join(srcDir, path), build.FindOnly)

	case p.isAbsPath(path):
		return nil, fmt.Errorf("invalid absolute import path %q", path)
	}
	if err != nil {
		return nil, err // err may be *build.NoGoError - return as is
	}

	// package unsafe is known to the type checker
	if bp.ImportPath == "unsafe" {
		return types.Unsafe, nil
	}

	// no need to re-import if the package was imported completely before
	pkg := p.typPkgs[bp.ImportPath]
	if pkg != nil {
		if pkg == &importing {
			return nil, fmt.Errorf("import cycle through package %q", bp.ImportPath)
		}
		if !pkg.Complete() {
			// Package exists but is not complete - we cannot handle this
			// at the moment since the source importer replaces the package
			// wholesale rather than augmenting it (see #19337 for details).
			// Return incomplete package with error (see #16088).
			return pkg, fmt.Errorf("reimported partially imported package %q", bp.ImportPath)
		}
		return pkg, nil
	}

	p.typPkgs[bp.ImportPath] = &importing
	defer func() {
		// clean up in case of error
		// TODO(gri) Eventually we may want to leave a (possibly empty)
		// package in the map in all cases (and use that package to
		// identify cycles). See also issue 16088.
		if p.typPkgs[bp.ImportPath] == &importing {
			p.typPkgs[bp.ImportPath] = nil
		}
	}()

	// collect package files
	bp, err = p.ctxt.ImportDir(bp.Dir, 0)
	if err != nil {
		return nil, err // err may be *build.NoGoError - return as is
	}
	var filenames []string
	filenames = append(filenames, bp.GoFiles...)
	filenames = append(filenames, bp.CgoFiles...)
	if p.IncludeTests != nil && p.IncludeTests(bp.ImportPath) {
		filenames = append(filenames, bp.TestGoFiles...)
	}

	files, err := p.parseFiles(bp.Dir, filenames, p.mode, nil)
	if err != nil {
		return nil, err
	}

	// type-check package files
	var firstHardErr error
	conf := types.Config{
		ParseFuncBodies: func(lbrace, rbrace token.Pos) bool {
			return false
		},
		FakeImportC: true,
		// continue type-checking after the first error
		Error: func(err error) {
			if firstHardErr == nil && !err.(types.Error).Soft {
				firstHardErr = err
			}
		},
		Importer: p,
		Sizes:    p.sizes,
	}
	pkg, err = conf.Check(bp.ImportPath, p.fset, files, p.info)
	if err != nil {
		// If there was a hard error it is possibly unsafe
		// to use the package as it may not be fully populated.
		// Do not return it (see also #20837, #20855).
		if firstHardErr != nil {
			pkg = nil
			err = firstHardErr // give preference to first hard error over any soft error
		}
		return pkg, fmt.Errorf("type-checking package %q failed (%v)", bp.ImportPath, err)
	}
	if firstHardErr != nil {
		// this can only happen if we have a bug in go/types
		panic("package is not safe yet no error was returned")
	}

	p.typPkgs[bp.ImportPath] = pkg
	return pkg, nil
}

func (p *Importer) parseFiles(dir string, filenames []string, mode parser.Mode, parseFuncBodies parser.InFuncBodies) ([]*ast.File, error) {
	open := p.ctxt.OpenFile // possibly nil

	files := make([]*ast.File, len(filenames))
	errors := make([]error, len(filenames))

	var wg sync.WaitGroup
	wg.Add(len(filenames))
	for i, filename := range filenames {
		go func(i int, filepath string) {
			defer wg.Done()
			file, cached := p.astPkgs.cachedFile(filepath)
			if cached {
				files[i], errors[i] = file, nil
			} else {
				if open != nil {
					src, err := open(filepath)
					if err != nil {
						errors[i] = fmt.Errorf("opening package file %s failed (%v)", filepath, err)
						return
					}
					files[i], errors[i] = parser.ParseFile(p.fset, filepath, src, mode, parseFuncBodies)
					src.Close() // ignore Close error - parsing may have succeeded which is all we need
				} else {
					// Special-case when ctxt doesn't provide a custom OpenFile and use the
					// parser's file reading mechanism directly. This appears to be quite a
					// bit faster than opening the file and providing an io.ReaderCloser in
					// both cases.
					// TODO(gri) investigate performance difference (issue #19281)
					files[i], errors[i] = parser.ParseFile(p.fset, filepath, nil, mode, parseFuncBodies)
				}
				if errors[i] == nil {
					p.astPkgs.cacheFile(filepath, files[i])
				}
			}
		}(i, p.joinPath(dir, filename))
	}
	wg.Wait()

	// if there are errors, return the first one for deterministic results
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// context-controlled file system operations

func (p *Importer) absPath(path string) (string, error) {
	// TODO(gri) This should be using p.ctxt.AbsPath which doesn't
	// exist but probably should. See also issue #14282.
	return filepath.Abs(path)
}

func (p *Importer) isAbsPath(path string) bool {
	if f := p.ctxt.IsAbsPath; f != nil {
		return f(path)
	}
	return filepath.IsAbs(path)
}

func (p *Importer) joinPath(elem ...string) string {
	if f := p.ctxt.JoinPath; f != nil {
		return f(elem...)
	}
	return filepath.Join(elem...)
}

func (p *Importer) readDir(path string) ([]os.FileInfo, error) {
	if f := p.ctxt.ReadDir; f != nil {
		return f(path)
	}
	return ioutil.ReadDir(path)
}

func (p *Importer) openFile(path string) ([]byte, error) {
	if f := p.ctxt.OpenFile; f != nil {
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

func (p *Importer) ParseFile(fileName string, parseFuncBodies parser.InFuncBodies) (*ast.File, error) {
	astFiles, err := p.parseFiles("", []string{fileName}, p.mode, parseFuncBodies)
	if err != nil {
		return nil, err
	}
	return astFiles[0], nil
}

func (p *Importer) ParseDir(dir string) ([]*ast.File, error) {
	list, err := p.readDir(dir)
	if err != nil {
		return nil, err
	}
	fileNames := make([]string, 0, len(list))
	for _, f := range list {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".go") && !strings.HasPrefix(f.Name(), ".") {
			fileNames = append(fileNames, f.Name())
		}
	}
	return p.parseFiles(dir, fileNames, p.mode, nil)
}

func (p *Importer) PathEnclosingInterval(fileName string, start, end token.Pos) []ast.Node {
	if f, ok := p.astPkgs.cachedFile(fileName); ok {
		nodes, _ := PathEnclosingInterval(f, start, end)
		return nodes
	}
	return []ast.Node{}
}

func (p *Importer) GetCachedPackage(pkgName string) (*ast.Package, bool) {
	return p.astPkgs.cachedPackage(pkgName)
}
