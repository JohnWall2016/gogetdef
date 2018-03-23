// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package importer

import (
	"bufio"
	"bytes"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// OverlayContext overlays a build.Context with additional files from
// a map. Files in the map take precedence over other files.
//
// In addition to plain string comparison, two file names are
// considered equal if their base names match and their directory
// components point at the same directory on the file system. That is,
// symbolic links are followed for directories, but not files.
//
// A common use case for OverlayContext is to allow editors to pass in
// a set of unsaved, modified files.
//
// Currently, only the Context.OpenFile function will respect the
// overlay. This may change in the future.
func OverlayContext(orig *build.Context, overlay map[string][]byte) *build.Context {
	// TODO(dominikh): Implement IsDir, HasSubdir and ReadDir

	rc := func(data []byte) (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(data)), nil
	}

	copy := *orig // make a copy
	ctxt := &copy
	ctxt.OpenFile = func(path string) (io.ReadCloser, error) {
		// Fast path: names match exactly.
		if content, ok := overlay[path]; ok {
			return rc(content)
		}

		// Slow path: check for same file under a different
		// alias, perhaps due to a symbolic link.
		for filename, content := range overlay {
			if sameFile(path, filename) {
				return rc(content)
			}
		}

		return OpenFile(orig, path)
	}
	ctxt.IsDir = func(path string) bool {
		if IsDir(orig, path) {
			return true
		}

		for filename, _ := range overlay {
			if _, ok := hasSubdir(path, filepath.Dir(filename)); ok {
				return true
			}
		}
		return false
	}
	ctxt.HasSubdir = func(root, dir string) (rel string, ok bool) {
		if rel, ok = HasSubdir(orig, root, dir); ok {
			return
		}
		return "", false
	}
	ctxt.ReadDir = func(dir string) (fis []os.FileInfo, err error) {
		fis, err = ReadDir(orig, dir)
		if err != nil {
			return
		}
		for filename, bytes := range overlay {
			if rel, ok := hasSubdir(dir, filename); ok {
				idx := strings.IndexRune(rel, filepath.Separator)
				if idx < 0 { // file
					fis = append(fis, &fileinfo{
						name: rel,
						size: int64(len(bytes)),
						dir:  false,
						mode: 0644,
					})
				} else { // dir
					fis = append(fis, &fileinfo{
						name: rel[:idx],
						dir:  true,
						mode: 0755,
					})
				}
			}
		}
		return
	}
	return ctxt
}

type fileinfo struct {
	name string
	size int64
	mode os.FileMode
	time time.Time
	dir  bool
}

func (fi *fileinfo) Name() string {
	return fi.name
}

func (fi *fileinfo) Size() int64 {
	return fi.size
}

func (fi *fileinfo) Mode() os.FileMode {
	return fi.mode
}

func (fi *fileinfo) ModTime() time.Time {
	return fi.time
}

func (fi *fileinfo) IsDir() bool {
	return fi.dir
}

func (fi *fileinfo) Sys() interface{} {
	return nil
}

// ParseOverlayArchive parses an archive containing Go files and their
// contents. The result is intended to be used with OverlayContext.
//
//
// Archive format
//
// The archive consists of a series of files. Each file consists of a
// name, a decimal file size and the file contents, separated by
// newlinews. No newline follows after the file contents.
func ParseOverlayArchive(archive io.Reader) (map[string][]byte, error) {
	overlay := make(map[string][]byte)
	r := bufio.NewReader(archive)
	for {
		// Read file name.
		filename, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break // OK
			}
			return nil, fmt.Errorf("reading archive file name: %v", err)
		}
		filename = filepath.Clean(strings.TrimSpace(filename))

		// Read file size.
		sz, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading size of archive file %s: %v", filename, err)
		}
		sz = strings.TrimSpace(sz)
		size, err := strconv.ParseUint(sz, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parsing size of archive file %s: %v", filename, err)
		}

		// Read file content.
		content := make([]byte, size)
		
		if _, err := io.ReadFull(r, content); err != nil {
			return nil, fmt.Errorf("reading archive file %s: %v", filename, err)
		}
		overlay[filename] = stripCR(content)
	}

	return overlay, nil
}

func stripCR(b []byte) []byte {
	c := make([]byte, len(b))
	i := 0
	for _, ch := range b {
		if ch != '\r' {
			c[i] = ch
			i++
		}
	}
	return c[:i]
}
