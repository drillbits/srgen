//    Copyright 2017 drillbits
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package srgen

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"
)

var servicesTmpl = `package {{.Package}}

var reg = &ServiceRegistry{}

// ServiceRegistry is an registry for services.
type ServiceRegistry struct {
	valid bool

{{- range .Services}}
	{{.}} {{.}}
{{- end}}
}

// Validate reports whether all services are fulfilled.
func (reg *ServiceRegistry) Validate() error {
	if reg.valid {
		return nil
	}

	var errs []string

{{- range .Services}}
	if reg.{{.}} == nil {
		errs = append(errs, "{{.}}")
	}
{{- end}}

	if len(errs) > 0 {
		return fmt.Errorf("nil service(s): %s", strings.Join(errs, ", "))
	}

	return nil
}

// Services returns the ServiceRegistry.
func Services() *ServiceRegistry {
	if err := reg.Validate(); err != nil {
		panic(err)
	}
	return reg
}
`

func Generate(files []string, outfile string) error {
	var pkg string
	var services []string
	for _, file := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		if pkg == "" {
			pkg = f.Name.Name
		} else if pkg != f.Name.Name {
			return fmt.Errorf("multiple packages: %s, %s", pkg, f.Name.Name)
		}

		ast.Inspect(f, func(node ast.Node) bool {
			d, ok := node.(*ast.GenDecl)
			if !ok || d.Tok != token.TYPE {
				return true
			}

			if !findTag(d, "+srgen") {
				return true
			}

			for _, spec := range d.Specs {
				t := spec.(*ast.TypeSpec)
				_, ok := t.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				services = append(services, t.Name.Name)
				return false
			}

			return true
		})
	}
	sort.Strings(services)

	if outfile == "" {
		outfile = "services.go"
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	t := template.Must(template.New("services").Parse(servicesTmpl))
	t.Execute(w, struct {
		Package  string
		Services []string
	}{
		Package:  pkg,
		Services: services,
	})
	w.Flush()

	b, err := imports.Process(outfile, buf.Bytes(), nil)
	if err != nil {
		return err
	}

	f, err := os.Create(outfile)
	if err != nil {
		return err
	}
	defer f.Close()

	w = bufio.NewWriter(f)
	defer w.Flush()
	w.Write(b)

	return nil
}

func findTag(d *ast.GenDecl, tag string) bool {
	if d.Doc == nil {
		return false
	}
	for _, c := range d.Doc.List {
		comment := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
		if strings.HasPrefix(comment, tag) {
			return true
		}
	}
	return false
}
