package asm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gasm/internal/arch"
	"gasm/internal/ast"
	"gasm/internal/format"
	"strings"
)

type Assembler struct {
	encoder arch.Encoder
	builder format.Builder
}

func NewAssembler(encoder arch.Encoder, builder format.Builder) *Assembler {
	return &Assembler{
		encoder: encoder,
		builder: builder,
	}
}

type AssemblyResult struct {
	Code     []byte
	Data     []byte
	Symbols  []format.Symbol
	Relocs   []format.Reloc
	Sections []format.Section
}

func (a *Assembler) Assemble(f *ast.File) (*AssemblyResult, error) {
	result := &AssemblyResult{}

	var codeBuf bytes.Buffer
	var dataBuf bytes.Buffer

	syms := make(map[string]format.Symbol)
	var relocs []format.Reloc

	currentSection := ".text"

	for _, it := range f.Items {
		switch n := it.(type) {
		case *ast.Label:
			if _, exists := syms[n.Name]; exists {
				return nil, fmt.Errorf("duplicate label: %s", n.Name)
			}

			var offset uint64
			if currentSection == ".text" {
				offset = uint64(codeBuf.Len())
			} else {
				offset = uint64(dataBuf.Len())
			}

			sym := format.Symbol{
				Name:    n.Name,
				Section: currentSection,
				Offset:  offset,
			}
			syms[n.Name] = sym

		case *ast.Directive:
			if len(n.Args) > 0 {
				switch n.Args[0] {
				case ".text", "text":
					currentSection = ".text"
				case ".data", "data":
					currentSection = ".data"
				}
			}

		case *ast.DataDecl:
			currentSection = ".data"
			for _, item := range n.Items {
				if item.IsStr {
					dataBuf.WriteString(item.Str)
				} else {
					switch v := item.Expr.(type) {
					case ast.NumberExpr:
						switch n.Kind {
						case "db":
							dataBuf.WriteByte(byte(v.Val))
						case "dw":
							var tmp [2]byte
							binary.LittleEndian.PutUint16(tmp[:], uint16(v.Val))
							dataBuf.Write(tmp[:])
						case "dd":
							var tmp [4]byte
							binary.LittleEndian.PutUint32(tmp[:], uint32(v.Val))
							dataBuf.Write(tmp[:])
						case "dq":
							var tmp [8]byte
							binary.LittleEndian.PutUint64(tmp[:], uint64(v.Val))
							dataBuf.Write(tmp[:])
						}
					case ast.IdentExpr:
						relocs = append(relocs, format.Reloc{
							Section: ".data",
							Offset:  uint64(dataBuf.Len()),
							Size:    8,
							Name:    v.Name,
							Kind:    int(arch.RelocAbs64),
						})
						dataBuf.Write(make([]byte, 8))
					}
				}
			}

		case *ast.Instruction:
			if currentSection == ".text" {
				code, err := a.encoder.EncodeInstruction(n)
				if err != nil {
					return nil, fmt.Errorf("line %d: %v", n.Line, err)
				}

				for _, op := range n.Operands {
					if lbl, ok := op.(ast.LabelOperand); ok {
						mn := strings.ToLower(n.Mnemonic)
						if mn == "jmp" || strings.HasPrefix(mn, "j") || mn == "call" {
							relocs = append(relocs, format.Reloc{
								Section: ".text",
								Offset:  uint64(codeBuf.Len() + 1),
								Size:    4,
								Name:    lbl.Name,
								Kind:    int(arch.RelocRel32),
							})
						}
					}
					if imm, ok := op.(ast.ImmOperand); ok {
						if ident, ok := imm.Val.(ast.IdentExpr); ok {
							relocs = append(relocs, format.Reloc{
								Section: ".text",
								Offset:  uint64(codeBuf.Len() + 2),
								Size:    8,
								Name:    ident.Name,
								Kind:    int(arch.RelocAbs64),
							})
						}
					}
					if lbl, ok := op.(ast.LabelOperand); ok {
						mn := strings.ToLower(n.Mnemonic)
						if mn == "mov" {
							relocs = append(relocs, format.Reloc{
								Section: ".text",
								Offset:  uint64(codeBuf.Len() + 2),
								Size:    8,
								Name:    lbl.Name,
								Kind:    int(arch.RelocAbs64),
							})
						}
					}
				}

				codeBuf.Write(code)
			}
		}
	}

	result.Code = codeBuf.Bytes()
	result.Data = dataBuf.Bytes()

	for _, sym := range syms {
		result.Symbols = append(result.Symbols, sym)
	}
	result.Relocs = relocs

	result.Sections = []format.Section{
		{Name: ".text", Data: result.Code},
		{Name: ".data", Data: result.Data},
	}

	return result, nil
}

func (a *Assembler) BuildBinary(result *AssemblyResult, outputPath string) ([]byte, error) {
	input := &format.BuilderInput{
		Sections: result.Sections,
		Symbols:  result.Symbols,
		Relocs:   result.Relocs,
		Arch:     int(a.encoder.Arch()),
		WordSize: a.encoder.WordSize(),
		Entry:    "_start",
	}

	bin, err := a.builder.Build(input)
	if err != nil {
		return nil, err
	}

	const pageSize = uint64(0x1000)
	const baseVaddr = uint64(0x400000)
	textFileOff := pageSize

	textVaddr := baseVaddr + textFileOff
	dataVaddr := textVaddr + uint64(len(result.Code))

	for _, r := range result.Relocs {
		var targetAddr uint64
		for _, s := range result.Symbols {
			if s.Name == r.Name {
				if s.Section == ".text" {
					targetAddr = textVaddr + s.Offset
				} else {
					targetAddr = dataVaddr + s.Offset
				}
				break
			}
		}

		if targetAddr == 0 {
			continue
		}

		targetAddrWithAdd := uint64(int64(targetAddr) + r.Addend)

		if r.Section == ".text" {
			offset := textFileOff + r.Offset
			if offset+uint64(r.Size) > uint64(len(bin)) {
				continue
			}

			switch arch.RelocKind(r.Kind) {
			case arch.RelocAbs64:
				binary.LittleEndian.PutUint64(bin[offset:offset+8], targetAddrWithAdd)
			case arch.RelocAbs32:
				binary.LittleEndian.PutUint32(bin[offset:offset+4], uint32(targetAddrWithAdd))
			case arch.RelocRel32:
				rel := int64(targetAddr) - int64(textVaddr) - int64(r.Offset) - 4 + r.Addend
				binary.LittleEndian.PutUint32(bin[offset:offset+4], uint32(int32(rel)))
			}
		} else {
			offset := textFileOff + uint64(len(result.Code)) + r.Offset
			if offset+uint64(r.Size) > uint64(len(bin)) {
				continue
			}

			if r.Size == 8 {
				binary.LittleEndian.PutUint64(bin[offset:offset+8], targetAddrWithAdd)
			} else if r.Size == 4 {
				binary.LittleEndian.PutUint32(bin[offset:offset+4], uint32(targetAddrWithAdd))
			}
		}
	}

	codeStart := textFileOff
	codeEnd := codeStart + uint64(len(result.Code))
	dataEnd := codeEnd + uint64(len(result.Data))

	if uint64(len(bin)) < dataEnd {
		newBin := make([]byte, dataEnd)
		copy(newBin, bin)
		bin = newBin
	}

	copy(bin[codeStart:codeEnd], result.Code)
	copy(bin[codeEnd:dataEnd], result.Data)

	return bin, nil
}
