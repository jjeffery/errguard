package main

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jjeffery/errguard/gen"
	"github.com/spf13/pflag"
)

var command struct {
	Filename string
	Types    []string
	Output   string
}

func main() {
	log.SetFlags(0)
	command.Filename = os.Getenv("GOFILE")
	pflag.StringVarP(&command.Filename, "file", "f", command.Filename, "Source file")
	pflag.StringVarP(&command.Output, "output", "o", defaultOutput(command.Filename), "Output file")
	pflag.Parse()
	command.Types = pflag.Args()
	if len(command.Types) == 0 {
		log.Fatal("no types specified")
	}
	if command.Filename == "" {
		log.Fatal("no file specified (-f or $GOFILE)")
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, command.Filename, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	model, err := gen.NewModel(file, command.Types)
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	if err := gen.DefaultTemplate.Execute(&buf, model); err != nil {
		log.Fatal(err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	var output io.Writer

	if command.Output == "" || command.Output == "-" {
		output = os.Stdout
	} else {
		outfile, err := os.Create(command.Output)
		if err != nil {
			log.Fatal(err)
		}
		defer outfile.Close()
		output = outfile
	}

	if _, err := output.Write(formatted); err != nil {
		log.Fatal(err)
	}

	//pretty.Print(model)
}

func defaultOutput(filename string) string {
	if filename == "" {
		return ""
	}
	output := strings.TrimSuffix(filename, filepath.Ext(filename))
	output = output + "_errguard.go"
	return output
}
