package asm

import (
	"os"
	"path/filepath"
	"testing"

	"gasm/internal/arch/x86_64"
	"gasm/internal/format/elf"
	"gasm/internal/parser"
)

func TestExamplesAssemble(t *testing.T) {
	examplesDir := filepath.Join("..", "..", "examples")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("ReadDir examples: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".asm" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(examplesDir, entry.Name())
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("Open example: %v", err)
			}
			defer f.Close()

			p := parser.New(f)
			astFile := p.ParseFile()
			if len(p.Errors) > 0 {
				t.Fatalf("parse errors: %v", p.Errors)
			}

			encoder := x86_64.NewEncoder()
			assembler := NewAssembler(encoder, elf.NewBuilder(encoder.Arch()))
			result, err := assembler.Assemble(astFile)
			if err != nil {
				t.Fatalf("Assemble failed: %v", err)
			}
			if len(result.Code) == 0 {
				t.Fatalf("example assembled to empty text section")
			}
		})
	}
}
