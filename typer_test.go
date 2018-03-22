package main

import (
	"path/filepath"
	"testing"
	"io/ioutil"
)

func TestFindTypeInFile(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "file1.go")
	if bytes, err := ioutil.ReadFile(testFile); err == nil {
		decl, pos, err := findTypeInFile(testFile, bytes, 88)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findTypeInFile(testFile, bytes, 97)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findTypeInFile(testFile, bytes, 179)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findTypeInFile(testFile, bytes, 117)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findTypeInFile(testFile, bytes, 197)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findTypeInFile(testFile, bytes, 273)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
	} else {
		t.Logf("%s", err.Error())
	}
}
