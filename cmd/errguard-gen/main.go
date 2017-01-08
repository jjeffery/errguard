package main

import (
	"go/parser"
	"go/token"
	"log"
	"os"

	"github.com/jjeffery/errguard/gen"
	"github.com/kr/pretty"
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

	if d := os.Getenv("DEBUGCD"); d != "" {
		pwd, _ := os.Getwd()
		log.Print(pwd)
		if err := os.Chdir(d); err != nil {
			log.Fatal(err)
		}
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

	model, err := gen.NewModel(file, []string{option.Type})
	if err != nil {
		log.Fatal(err)
	}
	pretty.Print(model)
	/*
		log.Println("interface", intf.Name)
		for _, method := range intf.Methods {
			log.Printf("%s(%s) (%s)", method.Name, method.ParamDecl, method.ResultDecl)
			log.Printf("(%s) (%s)", method.ArgNames, method.ResultNames)
			log.Printf("Error var = %q", method.ErrorVar)
			log.Printf("Context expr = %q", method.ContextExpr)
		}*/

}
