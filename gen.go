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
	{{- range .Imports}}
	{{if .Using}}{{.Name}} {{.Value}}{{end}}
	{{- end}}
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

{{- range $_, $s := .Services}}
// {{$s.Name}}Mock implements {{$s.Name}} for mocking.
type {{$s.Name}}Mock struct {
	{{- range $_, $m := $s.Methods}}
	{{- range $i, $r := $m.Results}}
	{{$m.Name}}Ret{{$i}} {{$r}}
	{{- end}}
	{{- end}}
}

{{- range $_, $m := $s.Methods}}
// {{$m.Name}} implements for mocking.
func (s *{{$s.Name}}Mock) {{$m.Name}}(
	{{- range $i, $p := $m.Params}}{{if $i}}, {{end}}{{$p}}{{end -}}
) (
	{{- range $i, $r := $m.Results}}{{if $i}}, {{end}}{{$r}}{{end -}}
) {
	return {{range $i, $r := $m.Results -}}
	{{- if $i}}, {{end -}}
	s.{{$m.Name}}Ret{{$i}}
	{{- end}}
}
{{- end}}

{{end}}
`

type Import struct {
	Name  string
	Value string
	Using bool
}

func (i *Import) DefaultName() string {
	s := strings.Replace(i.Value, "\"", "", -1)
	ss := strings.Split(s, "/")
	return ss[len(ss)-1]
}

type Service struct {
	Name    string
	Methods []*ServiceMethod
}

type ServiceMethod struct {
	Name    string
	Params  []string
	Results []string
}

func Generate(files []string, outfile string) error {
	var pkg string
	var imports []*Import
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
			if !ok {
				return true
			}

			// imports
			if d.Tok == token.IMPORT {
				for _, spec := range d.Specs {
					v, ok := spec.(*ast.ImportSpec)
					if !ok {
						continue
					}
					imports = append(imports, imp(v))
				}
				return true
			}

			// type or not
			if d.Tok != token.TYPE {
				return true
			}
			// tag
			if !findTag(d, "+srgen") {
				return true
			}

			for _, spec := range d.Specs {
				if !isInterface(spec) {
					continue
				}
				t := spec.(*ast.TypeSpec)
				svc := &Service{Name: t.Name.Name}

				i := t.Type.(*ast.InterfaceType)
				for _, f := range i.Methods.List {
					var fnName string
					for _, n := range f.Names {
						fnName = n.Name
					}
					if fnName == "" {
						continue
					}
					m := &ServiceMethod{Name: fnName}
					svc.Methods = append(svc.Methods, m)

					fn, ok := f.Type.(*ast.FuncType)
					if !ok {
						continue
					}

					for _, f := range fn.Params.List {
						s := fieldString(imports, f.Type)
						if s != "" {
							m.Params = append(m.Params, s)
						}
					}

					for _, f := range fn.Results.List {
						s := fieldString(imports, f.Type)
						if s != "" {
							m.Results = append(m.Results, s)
						}
					}
				}
				services = append(services, svc)
			}

			return true
		})
	}
	sort.SliceStable(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	uniq := make([]*Import, 0, len(imports))
	m := map[string]bool{}
	for _, i := range imports {
		k := fmt.Sprintf("%s%s", i.Name, i.Value)
		if !m[k] {
			m[k] = true
			uniq = append(uniq, i)
		}
	}
	imports = uniq

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	t := template.Must(template.New("services").Parse(servicesTmpl))
	t.Execute(w, struct {
		Package  string
		Imports  []*Import
		Services []*Service
	}{
		Package:  pkg,
		Imports:  imports,
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

func imp(spec *ast.ImportSpec) *Import {
	var name string
	if spec.Name != nil {
		name = spec.Name.Name
	}
	return &Import{
		Name:  name,
		Value: spec.Path.Value,
	}
}

func usingImp(imports []*Import, name string) {
	for _, i := range imports {
		if i.DefaultName() == name {
			i.Using = true
		}
	}
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

func isInterface(spec ast.Spec) bool {
	t, ok := spec.(*ast.TypeSpec)
	if !ok {
		return false
	}
	if _, ok := t.Type.(*ast.InterfaceType); !ok {
		return false
	}
	return true
}

func fieldString(imports []*Import, expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.ArrayType:
		s := fieldString(imports, v.Elt)
		if s != "" {
			return "[]" + s
		}
	case *ast.StarExpr:
		s := fieldString(imports, v.X)
		if s != "" {
			return "*" + s
		}
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		if x, ok := v.X.(*ast.Ident); ok {
			usingImp(imports, x.Name)
			return fmt.Sprintf("%s.%s", x.Name, v.Sel.Name)
		}
	}
	return ""
}
