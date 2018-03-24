package main

import (
	"path/filepath"
	"testing"
	"runtime"
	"bytes"
	"io/ioutil"
	"fmt"
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
	decl, pos, err := findDeclare(testFile, 169, nil)//169)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
	decl, pos, err = findDeclare(testFile, 177, nil)//177)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
	decl, pos, err = findDeclare(testFile, 146, nil)//146)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
	decl, pos, err = findDeclare(testFile, 142, nil)//146)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
	decl, pos, err = findDeclare(testFile, 29, nil)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
	decl, pos, err = findDeclare(testFile, 203, nil)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
}

func TestOverlayArchive(t *testing.T) {
	var buf bytes.Buffer
	testFile := filepath.Join(getTestDataDir(), "file2.go")
	s := testFile + "\n"
	buf.Write([]byte(s))
	buf2, _ := ioutil.ReadFile(testFile)
	s = fmt.Sprintf("%d\n", len(buf2))
	buf.Write([]byte(s))
	buf.Write(buf2)

	decl, pos, err := findDeclare(testFile, 169, &buf)
	if err == nil {
		t.Logf("%s, %s", decl, pos)
	} else {
		t.Logf("%s", err.Error())
	}
}
