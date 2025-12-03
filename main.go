package main

import (
	"fmt"
	"gasm/debug"
	"gasm/internal/ast/x86_64"
	"gasm/internal/builder"
	"os"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: nasm_parser <file.asm> <output>\n")
	os.Exit(2)
}

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	fn := os.Args[1]
	out := os.Args[2]
	f, err := os.Open(fn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	p := x86_64.NewParser(f)
	ast := p.ParseFile()
	if len(p.Errors) > 0 {
		fmt.Println("Warnings/Errors:")
		for _, e := range p.Errors {
			fmt.Println(" -", e)
		}
	}

	x86_64.PrintAST(ast)

	err = builder.AssembleAndBuildElf(out, ast)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("check sum is", debug.CheckSum(out))
}
