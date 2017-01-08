package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"

	"github.com/spf13/pflag"
)

var option struct {
	Type string
}

func main() {
	log.SetFlags(0)

	pflag.StringVarP(&option.Type, "type", "t", "", "Interface type for errguard implementation")
	pflag.Parse()
	if option.Type == "" {
		log.Fatal("missing --type option")
	}

	log.Println("environment:")
	for _, v := range []string{"GOOS", "GOARCH", "GOFILE", "GOLINE", "GOPACKAGE", "DOLLAR"} {
		log.Printf("    %s=%s", v, os.Getenv(v))
	}
	log.Println("options:")
	log.Printf("    Type=%s", option.Type)

	fileName := os.Getenv("GOFILE")
	if fileName == "" {
		log.Fatal("missing GOFILE")
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, fileName, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == option.Type {
						_, ok := typeSpec.Type.(*ast.InterfaceType)
						if !ok {
							log.Fatalf("Type %s is not an interface", option.Type)
						}
						/*
							for _, field := range interfaceType.Methods.List {
								for _, ident := range field.Names {
									log.Println(ident.Name)
								}
							}
						*/
						ast.Print(fset, typeSpec)
					}
				}
			}
		}
	}

}
