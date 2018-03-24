// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package importer

import (
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// -- Effective methods of file system interface -------------------------

// (go/build.Context defines these as methods, but does not export them.)

// hasSubdir calls ctxt.HasSubdir (if not nil) or else uses
// the local file system to answer the question.
func HasSubdir(ctxt *build.Context, root, dir string) (rel string, ok bool) {
	if f := ctxt.HasSubdir; f != nil {
		return f(root, dir)
	}

	// Try using paths we received.
	if rel, ok = hasSubdir(root, dir); ok {
		return
	}

	// Try expanding symlinks and comparing
	// expanded against unexpanded and
	// expanded against expanded.
	rootSym, _ := filepath.EvalSymlinks(root)
	dirSym, _ := filepath.EvalSymlinks(dir)

	if rel, ok = hasSubdir(rootSym, dir); ok {
		return
	}
	if rel, ok = hasSubdir(root, dirSym); ok {
		return
	}
	return hasSubdir(rootSym, dirSym)
}

func hasSubdir(root, dir string) (rel string, ok bool) {
	const sep = string(filepath.Separator)
	root = filepath.Clean(root)
	if !strings.HasSuffix(root, sep) {
		root += sep
	}

	dir = filepath.Clean(dir)
	if !strings.HasPrefix(dir, root) {
		return "", false
	}

	return filepath.ToSlash(dir[len(root):]), true
}

// FileExists returns true if the specified file exists,
// using the build context's file system interface.
func FileExists(ctxt *build.Context, path string) bool {
	if ctxt.OpenFile != nil {
		r, err := ctxt.OpenFile(path)
		if err != nil {
			return false
		}
		r.Close() // ignore error
		return true
	}
	_, err := os.Stat(path)
	return err == nil
}

// OpenFile behaves like os.Open,
// but uses the build context's file system interface, if any.
func OpenFile(ctxt *build.Context, path string) (io.ReadCloser, error) {
	if ctxt.OpenFile != nil {
		return ctxt.OpenFile(path)
	}
	return os.Open(path)
}

// IsAbsPath behaves like filepath.IsAbs,
// but uses the build context's file system interface, if any.
func IsAbsPath(ctxt *build.Context, path string) bool {
	if ctxt.IsAbsPath != nil {
		return ctxt.IsAbsPath(path)
	}
	return filepath.IsAbs(path)
}

// JoinPath behaves like filepath.Join,
// but uses the build context's file system interface, if any.
func JoinPath(ctxt *build.Context, path ...string) string {
	if ctxt.JoinPath != nil {
		return ctxt.JoinPath(path...)
	}
	return filepath.Join(path...)
}

// IsDir behaves like os.Stat plus IsDir,
// but uses the build context's file system interface, if any.
func IsDir(ctxt *build.Context, path string) bool {
	if ctxt.IsDir != nil {
		return ctxt.IsDir(path)
	}
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// ReadDir behaves like ioutil.ReadDir,
// but uses the build context's file system interface, if any.
func ReadDir(ctxt *build.Context, path string) ([]os.FileInfo, error) {
	if ctxt.ReadDir != nil {
		return ctxt.ReadDir(path)
	}
	return ioutil.ReadDir(path)
}

// SplitPathList behaves like filepath.SplitList,
// but uses the build context's file system interface, if any.
func SplitPathList(ctxt *build.Context, s string) []string {
	if ctxt.SplitPathList != nil {
		return ctxt.SplitPathList(s)
	}
	return filepath.SplitList(s)
}

// SameFile returns true if x and y have the same basename and denote
// the same file.
//
func SameFile(x, y string) bool {
	if path.Clean(x) == path.Clean(y) {
		return true
	}
	if filepath.Base(x) == filepath.Base(y) { // (optimisation)
		if xi, err := os.Stat(x); err == nil {
			if yi, err := os.Stat(y); err == nil {
				return os.SameFile(xi, yi)
			}
		}
	}
	return false
}
