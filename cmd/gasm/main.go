package main

import (
	"fmt"
	"gasm/internal/arch"
	"gasm/internal/arch/x86_64"
	"gasm/internal/asm"
	"gasm/internal/format"
	"gasm/internal/format/elf"
	"gasm/internal/format/pe"
	"gasm/internal/parser"
	"os"
	"path/filepath"
	"strings"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: gasm [options] <input.asm> <output>\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  -arch <arch>      Target architecture: x86, x86_64, arm, arm64 (default: x86_64)\n")
	fmt.Fprintf(os.Stderr, "  -format <format>  Output format: elf, pe (default: elf)\n")
	fmt.Fprintf(os.Stderr, "  -o <file>         Output file\n")
	os.Exit(2)
}

func main() {
	if len(os.Args) < 3 {
		usage()
	}

	var inputFile, outputFile string
	var targetArch = arch.ArchX86_64
	var targetFormat = format.FormatELF

	i := 1
	for i < len(os.Args) {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "-arch":
				if i+1 >= len(os.Args) {
					fmt.Fprintf(os.Stderr, "Error: -arch requires argument\n")
					os.Exit(1)
				}
				targetArch = arch.ParseArch(os.Args[i+1])
				if targetArch == arch.ArchUnknown {
					fmt.Fprintf(os.Stderr, "Error: unknown architecture: %s\n", os.Args[i+1])
					os.Exit(1)
				}
				i += 2
			case "-format":
				if i+1 >= len(os.Args) {
					fmt.Fprintf(os.Stderr, "Error: -format requires argument\n")
					os.Exit(1)
				}
				targetFormat = format.ParseFormat(os.Args[i+1])
				if targetFormat == format.FormatUnknown {
					fmt.Fprintf(os.Stderr, "Error: unknown format: %s\n", os.Args[i+1])
					os.Exit(1)
				}
				i += 2
			case "-o":
				if i+1 >= len(os.Args) {
					fmt.Fprintf(os.Stderr, "Error: -o requires argument\n")
					os.Exit(1)
				}
				outputFile = os.Args[i+1]
				i += 2
			default:
				fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", arg)
				os.Exit(1)
			}
		} else {
			if inputFile == "" {
				inputFile = arg
			} else if outputFile == "" {
				outputFile = arg
			}
			i++
		}
	}

	if inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: no input file specified\n")
		os.Exit(1)
	}
	if outputFile == "" {
		base := filepath.Base(inputFile)
		ext := filepath.Ext(base)
		outputFile = base[:len(base)-len(ext)]
	}

	f, err := os.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	p := parser.New(f)
	astFile := p.ParseFile()

	if len(p.Errors) > 0 {
		fmt.Println("Errors:")
		for _, e := range p.Errors {
			fmt.Println(" -", e)
		}
		if len(p.Errors) > 0 {
			os.Exit(1)
		}
	}

	var encoder arch.Encoder
	switch targetArch {
	case arch.ArchX86_64:
		encoder = x86_64.NewEncoder()
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported architecture: %s\n", targetArch)
		os.Exit(1)
	}

	var builder format.Builder
	switch targetFormat {
	case format.FormatELF:
		builder = elf.NewBuilder(targetArch)
	case format.FormatPE:
		builder = pe.NewBuilder(targetArch)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format: %s\n", targetFormat)
		os.Exit(1)
	}

	assembler := asm.NewAssembler(encoder, builder)

	result, err := assembler.Assemble(astFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	bin, err := assembler.BuildBinary(result, outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	outExt := builder.Extension()
	outPath := outputFile
	if outExt != "" && !strings.HasSuffix(outPath, outExt) {
		outPath = outPath + outExt
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	if _, err := outFile.Write(bin); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := outFile.Chmod(0755); err != nil {
	}

	fmt.Printf("Assembled %s -> %s (%s, %s)\n", inputFile, outPath, targetArch, targetFormat)
}
