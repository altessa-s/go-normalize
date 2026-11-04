// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/altessa-s/go-normalize"

func TestArchitecture(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	packages := map[string]bool{}

	err = filepath.WalkDir(root, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			if file != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "phone" || entry.Name() == "devtools" || entry.Name() == "docs") {
				return filepath.SkipDir
			}

			if file != root {
				if _, statErr := os.Stat(filepath.Join(file, "go.mod")); statErr == nil {
					return filepath.SkipDir
				}
			}

			return nil
		}

		if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			return nil
		}

		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.AllErrors)
		if err != nil {
			return err
		}

		dir, err := filepath.Rel(root, filepath.Dir(file))
		if err != nil {
			return err
		}

		dir = filepath.ToSlash(dir)
		if dir == "." {
			dir = ""
		}

		packages[dir] = true

		for _, im := range f.Imports {
			importPath, err := strconv.Unquote(im.Path.Value)
			if err != nil {
				return err
			}

			if foreign(importPath) {
				t.Errorf("%s has forbidden dependency %s", dir, importPath)
			}

			if importPath == "github.com/nyaruka/phonenumbers" {
				t.Errorf("root must not import phonenumbers")
			}

			switch dir {
			case "":
				if !stdlib(importPath) && importPath != modulePath+"/modifiers" {
					t.Errorf("root domain imports outer package %s", importPath)
				}
			case "modifiers":
				if !stdlib(importPath) {
					t.Errorf("modifiers imports external package %s", importPath)
				}
			}
		}

		for _, decl := range f.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				if decl.Recv == nil && decl.Name.Name == "init" {
					t.Errorf("init function in %s", file)
				}

				if decl.Recv == nil && strings.HasPrefix(decl.Name.Name, "With") && ast.IsExported(decl.Name.Name) {
					t.Errorf("handwritten option constructor %s in %s", decl.Name.Name, file)
				}
			case *ast.GenDecl:
				if decl.Tok == token.VAR {
					for _, spec := range decl.Specs {
						value, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}

						for _, name := range value.Names {
							if ast.IsExported(name.Name) && !strings.HasPrefix(name.Name, "Err") {
								t.Errorf("exported mutable variable %s in %s", name.Name, file)
							}
						}
					}
				}

				for _, spec := range decl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !ast.IsExported(typeSpec.Name.Name) || typeSpec.Name.Name != "Config" {
						continue
					}

					if _, ok := typeSpec.Type.(*ast.StructType); ok {
						t.Errorf("public Config struct in %s", file)
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for dir := range packages {
		rel := dir
		if rel == "" {
			rel = "."
		}

		if _, err := os.Stat(filepath.Join(root, rel, "doc.go")); err != nil {
			t.Errorf("package %s has no doc.go", dir)
		}

		if _, err := os.Stat(filepath.Join(root, rel, "README.md")); err != nil {
			t.Errorf("package %s has no README.md", dir)
		}
	}
}

// foreign reports whether importPath belongs to another altessa-s module; this
// repository depends on none of them.
func foreign(importPath string) bool {
	return strings.HasPrefix(importPath, "github.com/altessa-s/") && importPath != modulePath && !strings.HasPrefix(importPath, modulePath+"/")
}

func stdlib(importPath string) bool {
	first, _, _ := strings.Cut(importPath, "/")
	return !strings.Contains(first, ".")
}
