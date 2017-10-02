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
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"text/template"
)

var servicesTmpl = `package {{.Package}}

import (
	"fmt"
	"strings"
)

var reg = &ServiceRegistry{}

// ServiceRegistry is an registry for services.
type ServiceRegistry struct {
	valid bool

{{- range .Services}}
	{{.Name}} {{.Name}}
{{- end}}
}

// Validate reports whether all services are fulfilled.
func (reg *ServiceRegistry) Validate() error {
	if reg.valid {
		return nil
	}

	var errs []string

{{- range .Services}}
	if reg.{{.Name}} == nil {
		errs = append(errs, "{{.Name}}")
	}
{{- end}}

	if len(errs) > 0 {
		return fmt.Errorf("nil service(s): %s", strings.Join(errs, ", "))
	}

	return nil
}

// InitServiceRegistry initializes the ServiceRegistry.
func InitServiceRegistry(
{{- range .Services}}
	{{.Name}} {{.Name}},
{{- end}}
) {
{{- range .Services}}
	reg.{{.Name}} = {{.Name}}
{{- end}}
}

// Services returns the ServiceRegistry.
func Services() *ServiceRegistry {
	return reg
}

// MustServices is like Services but panics if all services are not fulfilled.
func MustServices() *ServiceRegistry {
	if err := reg.Validate(); err != nil {
		panic(err)
	}
	return reg
}
`

type Service struct {
	Name string
}

func Generate(files []string, outfile string) error {
	var pkg string
	var services []*Service
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
				svc := &Service{Name: t.Name.Name}
				services = append(services, svc)
				return false
			}

			return true
		})
	}
	sort.SliceStable(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	if outfile == "" {
		outfile = "services.go"
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	t := template.Must(template.New("services").Parse(servicesTmpl))
	t.Execute(w, struct {
		Package  string
		Services []*Service
	}{
		Package:  pkg,
		Services: services,
	})
	w.Flush()

	b, err := format.Source(buf.Bytes())
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
