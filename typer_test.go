package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
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

func TestBuiltinType(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "file3.go")
	def, err := findDefinition(testFile, 77, nil)
	if err == nil {
		t.Logf("%s, %s", def.decl, def.pos)
	} else {
		t.Logf("%s", err.Error())
	}
}

func TestFindDeclare(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "file2.go")
	def, err := findDefinition(testFile, 169, nil)
	if err == nil {
		t.Logf("%s, %s", def.decl, def.pos)
	} else {
		t.Logf("%s", err.Error())
	}
	def, err = findDefinition(testFile, 177, nil)
	if err == nil {
		t.Logf("%s, %s", def.decl, def.pos)
	} else {
		t.Logf("%s", err.Error())
	}
	def, err = findDefinition(testFile, 146, nil)
	if err == nil {
		t.Logf("%s, %s", def.decl, def.pos)
	} else {
		t.Logf("%s", err.Error())
	}
	def, err = findDefinition(testFile, 142, nil)
	if err == nil {
		t.Logf("%s, %s", def.decl, def.pos)
	} else {
		t.Logf("%s", err.Error())
	}
	def, err = findDefinition(testFile, 29, nil)
	if err == nil {
		t.Logf("%s, %s", def.decl, def.pos)
	} else {
		t.Logf("%s", err.Error())
	}
	def, err = findDefinition(testFile, 203, nil)
	if err == nil {
		t.Logf("%s, %s", def.decl, def.pos)
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

	def, err := findDefinition(testFile, 169, &buf)
	if err == nil {
		t.Logf("%s, %s", def.decl, def.pos)
	} else {
		t.Logf("%s", err.Error())
	}
}
