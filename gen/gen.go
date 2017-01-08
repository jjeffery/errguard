// Package gen generates code for errguard.
package gen

import (
	"fmt"
	"go/ast"
	"strconv"
	"strings"

	"github.com/jjeffery/stringset"
)

// Interface contains information about a single interface needed by the template
type Interface struct {
	TypeSpec      *ast.TypeSpec
	InterfaceType *ast.InterfaceType
	Name          string
	Methods       []*Method
}

// Method contains information about a single method needed by the template.
type Method struct {
	Interface   *Interface
	Field       *ast.Field
	Name        string
	ArgNames    string // Comma separated list of argument names
	ParamDecl   string // Parameters and types for method declaration
	ReturnNames string // Comma separated list of returns names
	ReturnDecl  string // Returns for method declaration
	Params      ParamList
	Returns     ParamList
}

type ParamList []*Param

func (pl ParamList) String() string {
	v := make([]string, len(pl))
	for i, p := range pl {
		v[i] = p.String()
	}
	return strings.Join(v, ", ")
}

type Param struct {
	Name     string // might be blank
	TypeName string
}

func (p *Param) String() string {
	if p.Name == "" {
		return p.TypeName
	}
	return p.Name + " " + p.TypeName
}

func MakeModel(file *ast.File, interfaceName string) (*Interface, error) {
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == interfaceName {
						interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
						if !ok {
							return nil, fmt.Errorf("type %s is not an interface", interfaceName)
						}
						return newInterface(typeSpec, interfaceType), nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("type %s not found", interfaceName)
}

func newInterface(typeSpec *ast.TypeSpec, interfaceType *ast.InterfaceType) *Interface {
	intf := &Interface{
		TypeSpec:      typeSpec,
		InterfaceType: interfaceType,
		Name:          typeSpec.Name.Name,
	}
	for _, field := range interfaceType.Methods.List {
		method := newMethod(intf, field)
		intf.Methods = append(intf.Methods, method)
	}

	return intf
}

func newMethod(intf *Interface, field *ast.Field) *Method {
	method := &Method{
		Interface: intf,
		Field:     field,
		Name:      field.Names[0].Name,
	}
	funcType := field.Type.(*ast.FuncType)

	// work out all the assigned names so that we can
	// assign unique ones for anonymous fields
	allNames := stringset.New()
	{
		for _, paramField := range funcType.Params.List {
			for _, ident := range paramField.Names {
				allNames.Add(ident.Name)
			}
		}
		for _, resultField := range funcType.Results.List {
			for _, ident := range resultField.Names {
				allNames.Add(ident.Name)
			}
		}
	}

	var argNames []string
	var paramDecls []string
	var resultNames []string
	var resultDecls []string

	for _, paramField := range funcType.Params.List {
		typeString := exprString(paramField.Type)
		var names []string
		for _, ident := range paramField.Names {
			names = append(names, ident.Name)
		}
		if len(names) == 0 {
			names = append(names, newParamName(allNames, typeString))
		}
		argNames = append(argNames, names...)
		paramDecl := fmt.Sprintf("%s %s", strings.Join(argNames, ", "), typeString)
		paramDecls = append(paramDecls, paramDecl)
	}
	for _, resultField := range funcType.Results.List {
		typeString := exprString(resultField.Type)
		var names []string
		for _, ident := range resultField.Names {
			names = append(names, ident.Name)
		}
		if len(names) == 0 {
			names = append(names, newParamName(allNames, typeString))
		}
		resultNames = append(resultNames, names...)
		resultDecl := fmt.Sprintf("%s %s", strings.Join(argNames, ", "), typeString)
		resultDecls = append(resultDecls, resultDecl)
	}

	method.ArgNames = strings.Join(argNames, ", ")
	method.ReturnNames = strings.Join(resultNames, ", ")
	method.ParamDecl = strings.Join(paramDecls, ", ")
	method.ReturnDecl = strings.Join(resultDecls, ", ")

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

func newParam(method *Method, field *ast.Field) *Param {
	param := &Param{
	//Method: method,
	//Field:  field,
	}
	return param
}

func exprString(t ast.Expr) string {
	if t == nil {
		return ""
	}
	switch v := t.(type) {
	case *ast.BadExpr:
		return "<bad-expr>"
	case *ast.Ident:
		return v.Name
	case *ast.Ellipsis:
		return fmt.Sprintf("...%s", exprString(v.Elt))
	case *ast.BasicLit:
		// does not appear in method declarations
		return v.Value
	case *ast.FuncLit:
		notExpecting("FuncLit")
	case *ast.CompositeLit:
		notExpecting("CompositeLit")
	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", exprString(v.X))
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", exprString(v.X), v.Sel.Name)
	case *ast.IndexExpr:
		return fmt.Sprintf("%s[%s]", exprString(v.X), exprString(v.Index))
	case *ast.SliceExpr:
		if v.Slice3 {
			return fmt.Sprintf("%s[%s:%s]", exprString(v.X), exprString(v.Low), exprString(v.High))
		}
		return fmt.Sprintf("%s[%s:%s:%s]", exprString(v.X), exprString(v.Low), exprString(v.High), exprString(v.Max))
	case *ast.TypeAssertExpr:
		notExpecting("TypeAssertExpr")
	case *ast.CallExpr:
		notExpecting("CallExpr")
	case *ast.StarExpr:
		return fmt.Sprintf("*%s", exprString(v.X))
	case *ast.UnaryExpr:
		return fmt.Sprintf("%s%s", v.Op.String(), exprString(v.X))
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", exprString(v.X), v.Op.String(), exprString(v.Y))
	case *ast.KeyValueExpr:
		return fmt.Sprintf("%s: %s", exprString(v.Key), exprString(v.Value))
	case *ast.ArrayType:
		return fmt.Sprintf("[%s]%s", exprString(v.Len), exprString(v.Elt))
	case *ast.StructType:
		notImplemented("StructType")
	case *ast.FuncType:
		notImplemented("FuncType")
	case *ast.InterfaceType:
		notImplemented("InterfaceType")
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprString(v.Key), exprString(v.Value))
	case *ast.ChanType:
		switch v.Dir {
		case ast.SEND:
			return fmt.Sprintf("chan<- %s", exprString(v.Value))
		case ast.RECV:
			return fmt.Sprintf("<-chan %s", exprString(v.Value))
		default:
			return fmt.Sprintf("chan %s", exprString(v.Value))
		}
	}

	panic(fmt.Sprintf("unknown ast.Expr: %v", t))
}

func notExpecting(nodeType string) {
	msg := fmt.Sprintf("not expecting node type of %s", nodeType)
	panic(msg)
}

func notImplemented(nodeType string) {
	msg := fmt.Sprintf("handling of node type not implemented: %s", nodeType)
	panic(msg)
}
