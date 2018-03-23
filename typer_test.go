package main

import (
	"path/filepath"
	"testing"
	"runtime"
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

func TestFindDeclare(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "file2.go")
	decl, pos, err := FindDeclare(testFile, 182, nil)//169)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
	decl, pos, err = FindDeclare(testFile, 189, nil)//177)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
	decl, pos, err = FindDeclare(testFile, 158, nil)//146)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
	decl, pos, err = FindDeclare(testFile, 154, nil)//146)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
	decl, pos, err = FindDeclare(testFile, 33, nil)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
	decl, pos, err = FindDeclare(testFile, 196, nil)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
}
