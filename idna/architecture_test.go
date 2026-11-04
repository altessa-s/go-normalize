// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package idna_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitecture(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	err = filepath.WalkDir(root, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() && file != root && strings.HasPrefix(entry.Name(), ".") {
			return filepath.SkipDir
		}

		if entry.IsDir() || !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			return nil
		}

		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.AllErrors)
		if err != nil {
			return err
		}

		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "init" {
				t.Errorf("init function in %s", file)
			}
		}

		for _, im := range f.Imports {
			importPath := strings.Trim(im.Path.Value, `"`)
			if foreign(importPath) {
				t.Errorf("forbidden dependency %s", importPath)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"doc.go", "README.md"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("idna module is missing %s", name)
		}
	}

	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}

	for line := range strings.Lines(string(mod)) {
		for field := range strings.FieldsSeq(line) {
			if foreign(field) {
				t.Errorf("idna module has a forbidden requirement %s", field)
			}
		}
	}
}

const modulePath = "github.com/altessa-s/go-normalize"

// foreign reports whether path belongs to another altessa-s module; this
// repository depends on none of them.
func foreign(path string) bool {
	return strings.HasPrefix(path, "github.com/altessa-s/") && path != modulePath && !strings.HasPrefix(path, modulePath+"/")
}
