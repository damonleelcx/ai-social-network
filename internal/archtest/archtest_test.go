// Package archtest makes the "scores stay hidden" rule a build failure rather than
// a promise. It reads the source, so it can't drift from what ships.
package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const hiddenPkg = "github.com/damonleelcx/ai-social-network/internal/hidden"

// Packages that face people. They must never hold a score, so they can't import
// the package that can read one.
var mustNotImportHidden = []string{"internal/httpapi", "internal/agent", "internal/dating", "internal/persona", "internal/profile", "internal/notify", "internal/chatstore", "internal/safety"}

func root(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "..", "..")
}

func goFiles(t *testing.T, dir string, f func(path string)) {
	t.Helper()
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			f(p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPeopleFacingPackagesCannotImportHidden(t *testing.T) {
	r := root(t)
	for _, pkg := range mustNotImportHidden {
		goFiles(t, filepath.Join(r, filepath.FromSlash(pkg)), func(p string) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, p, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range f.Imports {
				path, _ := strconv.Unquote(imp.Path.Value)
				if path == hiddenPkg {
					t.Errorf("%s imports internal/hidden; it faces people and must not be able to read a score", p)
				}
			}
		})
	}
}

// The schema has one door. Anyone writing "hidden." in SQL elsewhere fails here.
func TestOnlyHiddenPackageNamesTheSchema(t *testing.T) {
	r := root(t)
	goFiles(t, filepath.Join(r, "internal"), func(p string) {
		rel, _ := filepath.Rel(r, p)
		if strings.HasPrefix(filepath.ToSlash(rel), "internal/hidden/") {
			return
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		src := string(raw)
		for _, tbl := range []string{"hidden.compat", "hidden.evaluations", "hidden.reputation", "hidden.taste", "hidden.signals", "hidden.search_packs"} {
			if strings.Contains(src, tbl) {
				t.Errorf("%s names %s", rel, tbl)
			}
		}
	})
}
