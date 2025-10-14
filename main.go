package main

import (
	"fmt"
	"os"
	"path/filepath"

	parserlang "compilador/bindings/go"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func main() {
	parser := sitter.NewParser()
	defer parser.Close()

	// Wrap the unsafe.Pointer from parserlang.Language()
	rawLang := parserlang.Language()
	lang := sitter.NewLanguage(rawLang)

	// Set the language on the parser
	e := parser.SetLanguage(lang)
	if e != nil {
		panic(fmt.Errorf("couldn't configure parser: %w", e))
	}

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: compilador <input.ctds>")
		os.Exit(1)
	}

	inputArg := os.Args[1]

	if filepath.Ext(inputArg) != ".ctds" {
		fmt.Fprintln(os.Stderr, "error: input file must have .ctds extension")
		os.Exit(1)
	}

	var code []byte
	var err error
	code, err = os.ReadFile(inputArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	// Parse the code
	tree := parser.Parse(code, nil)
	defer tree.Close()

	// Get the root node
	root := tree.RootNode()

	if root.HasError() {
		fmt.Fprintf(os.Stderr, "could not parse file %s: syntax error\n", inputArg)

		os.Exit(1)
	}

	ast, err := BuildAST(root, code)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build error: %v\n", err)
		os.Exit(1)
	}

	// Run semantic analysis
	if ast != nil {
		if err := Analyze(ast); err != nil {
			fmt.Fprintf(os.Stderr, "semantic error: %v\n", err)
			os.Exit(1)
		}
	}

	// Prepare result directory and base filename
	base := inputArg[:len(inputArg)-len(filepath.Ext(inputArg))]
	baseName := filepath.Base(base)
	resultsDir := "result"
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating results directory: %v\n", err)
		os.Exit(1)
	}

	// Write AST (semantic stage) to .sem file in result folder
	semPath := filepath.Join(resultsDir, baseName+".sem")
	if ast != nil {
		if err := os.WriteFile(semPath, []byte(ast.String()), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing AST output: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("AST written to:", semPath)
	}

	// Pretty-print the syntax tree and write to .sint file in result folder
	output := []byte(root.ToSexp())
	outputPath := filepath.Join(resultsDir, baseName+".sint")
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Output written to:", outputPath)
}
