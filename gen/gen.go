// Package gen generates code for errguard.
package gen

import (
	"fmt"
	"go/ast"
	"path"
	"strconv"
	"strings"
	"text/template"

	"github.com/jjeffery/stringset"
)

// Model contains all of the information required to generate the file.
type Model struct {
	Package    string
	Imports    []*Import
	Interfaces []*Interface
	// TODO: functions
}

// Import describes a single import line required for the generated file.
type Import struct {
	Name string // Local name, or blank
	Path string
}

func (imp *Import) String() string {
	if imp.Name != "" {
		return fmt.Sprintf("%s %s", imp.Name, imp.Path)
	}
	return imp.Path
}

// Interface contains information about a single interface needed by the template
type Interface struct {
	Name    string
	Methods []*Method
}

// Method contains information about a single method needed by the template.
type Method struct {
	Interface   *Interface
	Name        string
	ArgNames    string // Comma separated list of input argument names
	ParamDecl   string // Parameters and types for method declaration
	ResultNames string // Comma separated list of result names
	ResultDecl  string // Results for method declaration
	ErrorVar    string // Name of the result error var
	ContextExpr string // Expression to use to obtain the context
}

type importResolver struct {
	imports []*ast.ImportSpec
	used    map[string]*Import
}

func (r *importResolver) Resolve(name string) *Import {
	if imp, ok := r.used[name]; ok {
		return imp
	}

	// strips the quotes from the import path and returns the base name
	pathBase := func(p string) string {
		return path.Base(strings.TrimPrefix(strings.TrimSuffix(p, `"`), `"`))
	}
	// strips the quotes from the import path and returns the base name without any extension
	// (good for import paths like "gopkg.in/xyz/abc.v1")
	pathBaseWithoutExtension := func(p string) string {
		p = pathBase(p)
		return strings.TrimSuffix(p, path.Ext(p))
	}

	tests := []func(*ast.ImportSpec) bool{
		// import has matching explicit name
		func(is *ast.ImportSpec) bool {
			return is.Name != nil && is.Name.Name == name
		},
		// import has matching import base name
		func(is *ast.ImportSpec) bool {
			if is.Name != nil {
				return false
			}
			return pathBase(is.Path.Value) == name
		},
		// import has matching import base name without extension
		func(is *ast.ImportSpec) bool {
			if is.Name != nil {
				return false
			}
			return pathBaseWithoutExtension(is.Path.Value) == name
		},
		// import base name contains the string somewhere
		func(is *ast.ImportSpec) bool {
			if is.Name != nil {
				return false
			}
			return strings.Contains(pathBase(is.Path.Value), name)
		},
	}
	// Search for an import whose name matches.
	for _, test := range tests {
		for _, importSpec := range r.imports {
			if test(importSpec) {
				imp := &Import{
					Path: importSpec.Path.Value,
				}
				if importSpec.Name != nil {
					imp.Name = importSpec.Name.Name
				}

				r.used[name] = imp
				return imp
			}
		}
	}
	return nil
}

func (r *importResolver) exprString(t ast.Expr) string {
	if t == nil {
		return ""
	}
	switch v := t.(type) {
	case *ast.BadExpr:
		return "<bad-expr>"
	case *ast.Ident:
		return v.Name
	case *ast.Ellipsis:
		return fmt.Sprintf("...%s", r.exprString(v.Elt))
	case *ast.BasicLit:
		// does not appear in method declarations
		return v.Value
	case *ast.FuncLit:
		notExpecting("FuncLit")
	case *ast.CompositeLit:
		notExpecting("CompositeLit")
	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", r.exprString(v.X))
	case *ast.SelectorExpr:
		r.Resolve(r.exprString(v.X))
		return fmt.Sprintf("%s.%s", r.exprString(v.X), v.Sel.Name)
	case *ast.IndexExpr:
		return fmt.Sprintf("%s[%s]", r.exprString(v.X), r.exprString(v.Index))
	case *ast.SliceExpr:
		if v.Slice3 {
			return fmt.Sprintf("%s[%s:%s]", r.exprString(v.X), r.exprString(v.Low), r.exprString(v.High))
		}
		return fmt.Sprintf("%s[%s:%s:%s]", r.exprString(v.X), r.exprString(v.Low), r.exprString(v.High), r.exprString(v.Max))
	case *ast.TypeAssertExpr:
		notExpecting("TypeAssertExpr")
	case *ast.CallExpr:
		notExpecting("CallExpr")
	case *ast.StarExpr:
		return fmt.Sprintf("*%s", r.exprString(v.X))
	case *ast.UnaryExpr:
		return fmt.Sprintf("%s%s", v.Op.String(), r.exprString(v.X))
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", r.exprString(v.X), v.Op.String(), r.exprString(v.Y))
	case *ast.KeyValueExpr:
		return fmt.Sprintf("%s: %s", r.exprString(v.Key), r.exprString(v.Value))
	case *ast.ArrayType:
		return fmt.Sprintf("[%s]%s", r.exprString(v.Len), r.exprString(v.Elt))
	case *ast.StructType:
		notImplemented("StructType")
	case *ast.FuncType:
		notImplemented("FuncType")
	case *ast.InterfaceType:
		notImplemented("InterfaceType")
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", r.exprString(v.Key), r.exprString(v.Value))
	case *ast.ChanType:
		switch v.Dir {
		case ast.SEND:
			return fmt.Sprintf("chan<- %s", r.exprString(v.Value))
		case ast.RECV:
			return fmt.Sprintf("<-chan %s", r.exprString(v.Value))
		default:
			return fmt.Sprintf("chan %s", r.exprString(v.Value))
		}
	}

	panic(fmt.Sprintf("unknown ast.Expr: %v", t))
}

func (r *importResolver) Imports() []*Import {
	var imports []*Import
	for _, imp := range r.used {
		imports = append(imports, imp)
	}
	return imports
}

func newImportResolver(imports []*ast.ImportSpec) (*importResolver, error) {
	for _, importSpec := range imports {
		if importSpec.Name != nil && importSpec.Name.Name == "." {
			return nil, fmt.Errorf("dot imports are not supported: . %v", importSpec.Path.Value)
		}
	}
	return &importResolver{
		imports: imports,
		used:    make(map[string]*Import),
	}, nil
}

// NewModel returns a model suitable for generating code from the information in
// the file AST and the list of names to generate code for. Each name should be
// the name of an interface or a function.
func NewModel(file *ast.File, names []string) (*Model, error) {
	model := &Model{
		Package: file.Name.Name,
	}
	ir, err := newImportResolver(file.Imports)
	if err != nil {
		return nil, err
	}

	nameSet := stringset.New(names...)
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					name := typeSpec.Name.Name
					if nameSet.Contains(name) {
						interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
						if !ok {
							return nil, fmt.Errorf("type %s is not an interface", name)
						}
						model.Interfaces = append(model.Interfaces, newInterface(ir, typeSpec, interfaceType))
					}
				}
			}
		}
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			// only interested in functions, not methods
			if funcDecl.Recv == nil {
				name := funcDecl.Name.Name
				if nameSet.Contains(name) {
					return nil, fmt.Errorf("found func %s, but functions are not supported yet", name)
				}
			}
		}
	}
	model.Imports = ir.Imports()

	// check for functions/methods that do not return an error
	{
		var missingErrs []string
		for _, intf := range model.Interfaces {
			for _, method := range intf.Methods {
				if method.ErrorVar == "" {
					missingErrs = append(missingErrs, fmt.Sprintf("%s.%s", intf.Name, method.Name))
				}
			}
		}
		if missingErrs != nil {
			return nil, fmt.Errorf("method does not return an error: %s", strings.Join(missingErrs, ", "))
		}
	}

	return model, nil
}

func newInterface(ir *importResolver, typeSpec *ast.TypeSpec, interfaceType *ast.InterfaceType) *Interface {
	intf := &Interface{
		Name: typeSpec.Name.Name,
	}
	for _, field := range interfaceType.Methods.List {
		method := newMethod(ir, intf, field)
		intf.Methods = append(intf.Methods, method)
	}

	return intf
}

func newMethod(ir *importResolver, intf *Interface, field *ast.Field) *Method {
	method := &Method{
		Interface: intf,
		Name:      field.Names[0].Name,
	}
	funcType := field.Type.(*ast.FuncType)

	// work out all the assigned names so that we can
	// assign unique ones for anonymous fields
	allNames := stringset.New()
	{
		if funcType.Params != nil {
			for _, paramField := range funcType.Params.List {
				for _, ident := range paramField.Names {
					allNames.Add(ident.Name)
				}
			}
		}
		if funcType.Results != nil {
			for _, resultField := range funcType.Results.List {
				for _, ident := range resultField.Names {
					allNames.Add(ident.Name)
				}
			}
		}
	}

	var argNames []string
	var paramDecls []string
	var resultNames []string
	var resultDecls []string
	var errorVar string
	var contextExpr string

	if funcType.Params.List != nil {
		for _, paramField := range funcType.Params.List {
			typeString := ir.exprString(paramField.Type)
			var names []string
			for _, ident := range paramField.Names {
				names = append(names, ident.Name)
			}
			if len(names) == 0 {
				names = append(names, newParamName(allNames, typeString))
			}
			argNames = append(argNames, names...)
			paramDecl := fmt.Sprintf("%s %s", strings.Join(names, ", "), typeString)
			paramDecls = append(paramDecls, paramDecl)
			if typeString == "context.Context" {
				contextExpr = names[0]
			}
		}
	}
	if funcType.Results != nil {
		for _, resultField := range funcType.Results.List {
			typeString := ir.exprString(resultField.Type)
			var names []string
			for _, ident := range resultField.Names {
				names = append(names, ident.Name)
			}
			if len(names) == 0 {
				names = append(names, newParamName(allNames, typeString))
			}
			resultNames = append(resultNames, names...)
			resultDecl := fmt.Sprintf("%s %s", strings.Join(names, ", "), typeString)
			resultDecls = append(resultDecls, resultDecl)
			if typeString == "error" {
				errorVar = names[0]
			}
		}
	}

	method.ArgNames = strings.Join(argNames, ", ")
	method.ResultNames = strings.Join(resultNames, ", ")
	method.ParamDecl = strings.Join(paramDecls, ", ")
	method.ResultDecl = strings.Join(resultDecls, ", ")
	method.ErrorVar = errorVar
	if contextExpr == "" {
		contextExpr = "context.TODO()"
	}
	method.ContextExpr = contextExpr

	return method
}

func newParamName(names stringset.Set, typeString string) string {
	var name string
	switch {
	case typeString == "error":
		name = "err"
	case strings.HasSuffix(typeString, ".Context"):
		name = "ctx"
	case strings.HasSuffix(typeString, "Input"):
		name = "input"
	case strings.HasSuffix(typeString, "Output"):
		name = "output"
	case strings.HasSuffix(typeString, "Request"):
		name = "request"
	case strings.HasSuffix(typeString, "Response"):
		name = "response"
	default:
		name = "a"
	}
	if !names.Contains(name) {
		names.Add(name)
		return name
	}
	for i := 1; ; i++ {
		namen := name + strconv.Itoa(i)
		if !names.Contains(namen) {
			names.Add(namen)
			return namen
		}
	}
}

func notExpecting(nodeType string) {
	msg := fmt.Sprintf("not expecting node type of %s", nodeType)
	panic(msg)
}

func notImplemented(nodeType string) {
	msg := fmt.Sprintf("handling of node type not implemented: %s", nodeType)
	panic(msg)
}

var DefaultTemplate = template.Must(template.New("defaultTemplate").Parse(`package {{.Package}}

// AUTOMATICALLY GENERATED -- DO NOT MODIFY

import ({{range .Imports}}
    {{.}}{{end}} 
    "github.com/jjeffery/errguard"
)

{{range .Interfaces}}
type guard{{.Name}} struct{
    inner {{.Name}}
}

func newGuard{{.Name}}(inner {{.Name}}) {{.Name}} {
    return &guard{{.Name}}{ inner: inner }
}
{{range .Methods}}

func (g *guard{{.Interface.Name}}) {{.Name}}({{.ParamDecl}}) ({{.ResultDecl}}) {
    var guard errguard.Guard
    {{.ErrorVar}} = guard.Run({{.ContextExpr}}, func() error {
        {{.ResultNames}} = g.inner.{{.Name}}({{.ArgNames}})
        return {{.ErrorVar}}
    })
    return {{.ResultNames}}
}
{{end}}
{{end}}`))
