package main

import (
	"path/filepath"
	"testing"
	"io/ioutil"
)

func TestFindTypeInFile(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "file2.go")
	if bytes, err := ioutil.ReadFile(testFile); err == nil {
		decl, pos, err := findTypeInFile(testFile, bytes, 169)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findTypeInFile(testFile, bytes, 177)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findTypeInFile(testFile, bytes, 146)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
	} else {
		t.Logf("%s", err.Error())
	}
}
