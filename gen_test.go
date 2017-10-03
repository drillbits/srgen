package srgen

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestGenerate(t *testing.T) {
	files := []string{
		"testdata/foo.go",
		"testdata/bar.go",
	}
	outfile := "testdata/services.go"
	err := Generate(files, outfile)
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, outfile, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	if f.Name.Name != "testdata" {
		t.Errorf("generated file's package = %s, but got %s", "testdata", f.Name.Name)
	}
}
