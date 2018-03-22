package main

import (
	"path/filepath"
	"testing"
	"runtime"
	"io/ioutil"
)

func getSrcCodeDir() string {
	if _, file, _, ok := runtime.Caller(0); ok {
		return filepath.Dir(file)
	} else {
		panic("can't get source code dir")
	}
}

func getTestDataDir() string {
	return filepath.Join(getSrcCodeDir(), "testdata")
}

func TestFindDeclareInFile(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "file1.go")
	if bytes, err := ioutil.ReadFile(testFile); err == nil {
		decl, pos, err := findDeclareInFile(testFile, bytes, 88)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findDeclareInFile(testFile, bytes, 97)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findDeclareInFile(testFile, bytes, 179)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findDeclareInFile(testFile, bytes, 117)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findDeclareInFile(testFile, bytes, 197)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
		decl, pos, err = findDeclareInFile(testFile, bytes, 273)
		if err == nil {
			t.Logf("%s, %s", decl, pos)
		} else {
			t.Logf("%s", err.Error())
		}
	} else {
		t.Logf("%s", err.Error())
	}
}
