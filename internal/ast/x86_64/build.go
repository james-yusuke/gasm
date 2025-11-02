package x86_64

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gasm/internal/elf"
	"os"
	"strings"
)

type Reloc struct {
	Section string
	Offset  uint64
	Size    int
	Name    string
	Addend  int64
	Kind    string
}

func AssembleAndBuildElf(path string, f *File) error {
	const baseVaddr = uint64(0x400000)
	const pageSize = uint64(64 + 56)
	textFileOff := pageSize

	var codeBuf bytes.Buffer
	var dataBuf bytes.Buffer

	type symRec struct {
		Section string
		Offset  uint64
	}

	syms := map[string]symRec{}

	var relocs []Reloc

	writeData := func(b []byte) {
		if _, err := dataBuf.Write(b); err != nil {
			panic(err)
		}
	}

	currentSection := ".data"

	regNameToID := func(name string) (int, bool) {
		n := strings.ToLower(name)
		switch n {
		case "rax", "eax":
			return 0, true
		case "rcx", "ecx":
			return 1, true
		case "rdx", "edx":
			return 2, true
		case "rbx", "ebx":
			return 3, true
		case "rsp", "esp":
			return 4, true
		case "rbp", "ebp":
			return 5, true
		case "rsi", "esi":
			return 6, true
		case "rdi", "edi":
			return 7, true
		case "r8":
			return 8, true
		case "r9":
			return 9, true
		case "r10":
			return 10, true
		case "r11":
			return 11, true
		case "r12":
			return 12, true
		case "r13":
			return 13, true
		case "r14":
			return 14, true
		case "r15":
			return 15, true
		}
		return 0, false
	}

	writeRex := func(regField, rmField int, needW bool) {
		var rex byte = 0x40
		if needW {
			rex |= 0x08
		}
		if regField >= 8 {
			rex |= 0x04
		}
		if rmField >= 8 {
			rex |= 0x01
		}
		codeBuf.WriteByte(rex)
	}

	writeRexIfNeeded := func(regs ...int) {
		var rex byte = 0x40
		rex |= 0x08
		var bbit byte = 0
		for _, r := range regs {
			if r >= 8 {
				bbit = 1
				break
			}
		}
		if bbit == 1 {
			rex |= 0x01
		}

		codeBuf.WriteByte(rex)
	}

	encodeMovRegImm64 := func(regID int, imm uint64) {
		writeRexIfNeeded(regID)
		op := byte(0xB8 | byte(regID&7))
		codeBuf.WriteByte(op)
		var tmp [8]byte
		binary.LittleEndian.PutUint64(tmp[:], imm)
		codeBuf.Write(tmp[:])
	}

	encodeMovRegImm32 := func(regID int, imm uint32) {
		op := byte(0xB8 | byte(regID&7))
		codeBuf.WriteByte(op)
		var tmp [4]byte
		binary.LittleEndian.PutUint32(tmp[:], imm)
		codeBuf.Write(tmp[:])
	}

	encodeXorRegReg := func(dstID, srcID int) {
		if dstID >= 8 || srcID >= 8 {
			writeRexIfNeeded(dstID, srcID)
		}
		codeBuf.WriteByte(0x31)
		modrm := byte(0xC0 | ((srcID & 7) << 3) | (dstID & 7))
		codeBuf.WriteByte(modrm)
	}

	writeModRM := func(regField, rmField int) {
		modrm := byte(0xC0 | ((regField & 7) << 3) | (rmField & 7))
		codeBuf.WriteByte(modrm)
	}

	encodeAddRegReg := func(dstID, srcID int) {
		writeRex(dstID, srcID, true)
		codeBuf.WriteByte(0x03)
		writeModRM(dstID, srcID)
	}

	encodeSubRegReg := func(dstID, srcID int) {
		writeRex(dstID, srcID, true)
		codeBuf.WriteByte(0x2B)
		writeModRM(dstID, srcID)
	}

	encodeAddRegImm32 := func(dstID int, imm32 uint32) {
		writeRex(0, dstID, true)
		codeBuf.WriteByte(0x81)
		modrm := byte(0xC0 | ((0 & 7) << 3) | (dstID & 7))
		codeBuf.WriteByte(modrm)
		var tmp [4]byte
		binary.LittleEndian.PutUint32(tmp[:], imm32)
		codeBuf.Write(tmp[:])
	}

	encodeAddRegImm8 := func(dstID int, imm8 int8) {
		writeRex(0, dstID, true)
		codeBuf.WriteByte(0x83)
		modrm := byte(0xC0 | ((0 & 7) << 3) | (dstID & 7))
		codeBuf.WriteByte(modrm)
		codeBuf.WriteByte(byte(imm8))
	}

	encodeSubRegImm32 := func(dstID int, imm32 uint32) {
		writeRex(5, dstID, true)
		codeBuf.WriteByte(0x81)
		modrm := byte(0xC0 | ((5 & 7) << 3) | (dstID & 7))
		codeBuf.WriteByte(modrm)
		var tmp [4]byte
		binary.LittleEndian.PutUint32(tmp[:], imm32)
		codeBuf.Write(tmp[:])
	}

	encodeSubRegImm8 := func(dstID int, imm8 int8) {
		writeRex(5, dstID, true)
		codeBuf.WriteByte(0x83)
		modrm := byte(0xC0 | ((5 & 7) << 3) | (dstID & 7))
		codeBuf.WriteByte(modrm)
		codeBuf.WriteByte(byte(imm8))
	}

	encodeDecReg := func(dstID int) {
		writeRex(1, dstID, true)
		codeBuf.WriteByte(0xFF)
		modrm := byte(0xC0 | ((1 & 7) << 3) | (dstID & 7))
		codeBuf.WriteByte(modrm)
	}

	encodeSyscall := func() {
		codeBuf.Write([]byte{0x0F, 0x05})
	}

	encodeRet := func() { codeBuf.WriteByte(0xC3) }

	for _, it := range f.Items {
		switch n := it.(type) {
		case *Label:
			if _, exists := syms[n.Name]; exists {
				return fmt.Errorf("duplicate label: %s", n.Name)
			}

			if currentSection == ".text" {
				syms[n.Name] = symRec{Section: ".text", Offset: uint64(codeBuf.Len())}
			} else {
				syms[n.Name] = symRec{Section: ".data", Offset: uint64(dataBuf.Len())}
			}
		case *Directive:
			switch n.Args[0] {
			case ".data":
				currentSection = ".data"
			case ".text":
				currentSection = ".text"
			}
		case *DataDecl:
			currentSection = ".data"
			for _, item := range n.Items {
				if item.IsStr {
					writeData([]byte(item.Str))
				} else {
					switch v := item.Expr.(type) {
					case NumberExpr:
						var tmp [8]byte
						binary.LittleEndian.PutUint64(tmp[:], uint64(v.Val))
						writeData(tmp[:])
					case IdentExpr:
						relocs = append(relocs, Reloc{Section: ".data", Offset: uint64(dataBuf.Len()), Size: 8, Name: v.Name, Kind: "abs64"})
						writeData(make([]byte, 8))
					default:
						writeData(make([]byte, 8))
					}
				}
			}
		case *Instruction:
			fmt.Println("n.Mnemonic", n.Mnemonic, currentSection)
			mn := strings.ToLower(n.Mnemonic)

			if currentSection == ".data" {
				if len(n.Operands) < 2 {
					return fmt.Errorf("%s needs at least a label and a string", mn)
				}

				lblOp, ok := n.Operands[0].(LabelOperand)
				if !ok {
					return fmt.Errorf("%s first operand must be a label: %T", mn, n.Operands[0])
				}

				strOp, ok := n.Operands[1].(StrOperand)
				if !ok {
					return fmt.Errorf("%s second operand must be a string: %T", mn, n.Operands[1])
				}

				if _, exists := syms[mn]; exists {
					return fmt.Errorf("duplicate label: %s", mn)
				}

				syms[mn] = symRec{Section: ".data", Offset: uint64(dataBuf.Len())}

				if _, err := dataBuf.Write([]byte(strOp.S)); err != nil {
					return err
				}

				if len(n.Operands) >= 3 {
					if im, ok := n.Operands[2].(ImmOperand); ok {
						if num, ok := im.Val.(NumberExpr); ok {
							desired := int(num.Val)
							written := int(dataBuf.Len()) - int(syms[lblOp.Name].Offset)
							if desired > written {
								dataBuf.Write(make([]byte, desired-written))
							}
						} else {
							return fmt.Errorf("%s size operand must be NumberExpr: %T", mn, im.Val)
						}
					}
				}

				// break
			} else if currentSection == ".text" {
				switch mn {
				case "mov":
					if len(n.Operands) != 2 {
						return fmt.Errorf("unsupported mov operand count: %d", len(n.Operands))
					}

					src := n.Operands[1]
					dst := n.Operands[0]
					if rd, ok := dst.(RegOperand); ok {
						regID, ok := regNameToID(rd.Name)
						if !ok {
							return fmt.Errorf("unknown reg: %s", rd.Name)
						}
						switch v := src.(type) {
						case ImmOperand:
							if num, ok := v.Val.(NumberExpr); ok {
								if num.Val >= -(1<<31) && num.Val < (1<<32) {
									encodeMovRegImm32(regID, uint32(num.Val))
								} else {
									encodeMovRegImm64(regID, uint64(num.Val))
								}
							} else if ident, ok := v.Val.(IdentExpr); ok {
								relocs = append(relocs, Reloc{Section: ".text", Offset: uint64(codeBuf.Len()) + 2, Size: 8, Name: ident.Name, Kind: "abs64"})
								writeRexIfNeeded(regID)
								codeBuf.WriteByte(0xB8 | byte(regID&7))
								codeBuf.Write(make([]byte, 8))
							} else {
								if nVal, err := evalToInt64(v.Val); err == nil {
									if nVal >= -(1<<31) && nVal < (1<<32) {
										encodeMovRegImm32(regID, uint32(nVal))
									} else {
										encodeMovRegImm64(regID, uint64(nVal))
									}
								} else {
									return fmt.Errorf("unsupported mov immediate expression: %T", v.Val)
								}
							}
						case LabelOperand:
							relocs = append(relocs, Reloc{Section: ".text", Offset: uint64(codeBuf.Len()) + 2, Size: 8, Name: v.Name, Kind: "abs64"})
							writeRexIfNeeded(regID)
							codeBuf.WriteByte(0xB8 | byte(regID&7))
							codeBuf.Write(make([]byte, 8))
						default:
							return fmt.Errorf("unsupported mov src operand type: %T", src)
						}
					} else {
						return fmt.Errorf("unsupported mov dst operand type: %T", dst)
					}
				case "xor":
					if len(n.Operands) != 2 {
						return fmt.Errorf("xor needs 2 operands")
					}

					if d, ok := n.Operands[0].(RegOperand); ok {
						if s, ok := n.Operands[1].(RegOperand); ok {
							did, _ := regNameToID(d.Name)
							sid, _ := regNameToID(s.Name)
							encodeXorRegReg(did, sid)
						} else {
							return fmt.Errorf("xor src not reg: %T", n.Operands[1])
						}
					} else {
						return fmt.Errorf("xor dst not reg: %T", n.Operands[0])
					}
				case "syscall":
					encodeSyscall()
				case "ret":
					encodeRet()
				case "add", "sub":
					if len(n.Operands) != 2 {
						return fmt.Errorf("%s needs 2 operands", mn)
					}
					dstOp := n.Operands[0]
					srcOp := n.Operands[1]

					rd, ok := dstOp.(RegOperand)
					if !ok {
						return fmt.Errorf("%s dst must be register: %T", mn, dstOp)
					}
					dstID, ok := regNameToID(rd.Name)
					if !ok {
						return fmt.Errorf("unknown reg: %s", rd.Name)
					}

					switch s := srcOp.(type) {
					case RegOperand:
						srcID, ok := regNameToID(s.Name)
						if !ok {
							return fmt.Errorf("unknown reg: %s", s.Name)
						}
						if mn == "add" {
							encodeAddRegReg(dstID, srcID)
						} else {
							encodeSubRegReg(dstID, srcID)
						}
					case ImmOperand:
						if num, ok := s.Val.(NumberExpr); ok {
							if num.Val >= -128 && num.Val <= 127 {
								if mn == "add" {
									encodeAddRegImm8(dstID, int8(num.Val))
								} else {
									encodeSubRegImm8(dstID, int8(num.Val))
								}
							} else {
								if mn == "add" {
									encodeAddRegImm32(dstID, uint32(num.Val))
								} else {
									encodeSubRegImm32(dstID, uint32(num.Val))
								}
							}
						} else if ident, ok := s.Val.(IdentExpr); ok {

							relocs = append(relocs, Reloc{Section: ".text", Offset: uint64(codeBuf.Len()) + 2, Size: 4, Name: ident.Name, Kind: "abs32"})
							if mn == "add" {
								writeRex(0, dstID, true)
								codeBuf.WriteByte(0x81)
								codeBuf.WriteByte(byte(0xC0 | ((0 & 7) << 3) | (dstID & 7)))
							} else {
								writeRex(5, dstID, true)
								codeBuf.WriteByte(0x81)
								codeBuf.WriteByte(byte(0xC0 | ((5 & 7) << 3) | (dstID & 7)))
							}
							codeBuf.Write(make([]byte, 4))
						} else {

							if nVal, err := evalToInt64(s.Val); err == nil {
								if nVal >= -128 && nVal <= 127 {
									if mn == "add" {
										encodeAddRegImm8(dstID, int8(nVal))
									} else {
										encodeSubRegImm8(dstID, int8(nVal))
									}
								} else {
									if mn == "add" {
										encodeAddRegImm32(dstID, uint32(nVal))
									} else {
										encodeSubRegImm32(dstID, uint32(nVal))
									}
								}
							} else {
								return fmt.Errorf("%s unsupported immediate operand: %T", mn, s.Val)
							}
						}
					case LabelOperand:

						relocs = append(relocs, Reloc{Section: ".text", Offset: uint64(codeBuf.Len()) + 2, Size: 4, Name: s.Name, Kind: "abs32"})
						if mn == "add" {
							writeRex(0, dstID, true)
							codeBuf.WriteByte(0x81)
							codeBuf.WriteByte(byte(0xC0 | ((0 & 7) << 3) | (dstID & 7)))
						} else {
							writeRex(5, dstID, true)
							codeBuf.WriteByte(0x81)
							codeBuf.WriteByte(byte(0xC0 | ((5 & 7) << 3) | (dstID & 7)))
						}
						codeBuf.Write(make([]byte, 4))
					default:
						return fmt.Errorf("%s unsupported src operand type: %T", mn, srcOp)
					}

				case "dec":

					if len(n.Operands) != 1 {
						return fmt.Errorf("dec needs 1 operand")
					}
					if rd, ok := n.Operands[0].(RegOperand); ok {
						id, ok := regNameToID(rd.Name)
						if !ok {
							return fmt.Errorf("unknown reg: %s", rd.Name)
						}
						encodeDecReg(id)
					} else {
						return fmt.Errorf("dec operand must be reg: %T", n.Operands[0])
					}
				case "jmp":

					if len(n.Operands) != 1 {
						return fmt.Errorf("jmp needs 1 operand")
					}
					labelOp, ok := n.Operands[0].(LabelOperand)
					if !ok {
						return fmt.Errorf("jmp operand must be label: %T", n.Operands[0])
					}

					codeBuf.WriteByte(0xE9)
					relocs = append(relocs, Reloc{
						Section: ".text",
						Offset:  uint64(codeBuf.Len()),
						Size:    4,
						Name:    labelOp.Name,
						Addend:  0,
						Kind:    "rel32",
					})
					codeBuf.Write(make([]byte, 4))

				case "je", "jz":

					if len(n.Operands) != 1 {
						return fmt.Errorf("%s needs 1 operand", n.Mnemonic)
					}
					labelOp, ok := n.Operands[0].(LabelOperand)
					if !ok {
						return fmt.Errorf("%s operand must be label: %T", n.Mnemonic, n.Operands[0])
					}

					codeBuf.Write([]byte{0x0F, 0x84})
					relocs = append(relocs, Reloc{
						Section: ".text",
						Offset:  uint64(codeBuf.Len()),
						Size:    4,
						Name:    labelOp.Name,
						Addend:  0,
						Kind:    "rel32",
					})
					codeBuf.Write(make([]byte, 4))

				case "jne", "jnz":
					if len(n.Operands) != 1 {
						return fmt.Errorf("%s needs 1 operand", n.Mnemonic)
					}
					labelOp, ok := n.Operands[0].(LabelOperand)
					if !ok {
						return fmt.Errorf("%s operand must be label: %T", n.Mnemonic, n.Operands[0])
					}

					codeBuf.Write([]byte{0x0F, 0x85})
					relocs = append(relocs, Reloc{
						Section: ".text",
						Offset:  uint64(codeBuf.Len()),
						Size:    4,
						Name:    labelOp.Name,
						Addend:  0,
						Kind:    "rel32",
					})
					codeBuf.Write(make([]byte, 4))

				case "jg":

					if len(n.Operands) != 1 {
						return fmt.Errorf("jg needs 1 operand")
					}
					labelOp, ok := n.Operands[0].(LabelOperand)
					if !ok {
						return fmt.Errorf("jg operand must be label: %T", n.Operands[0])
					}
					codeBuf.Write([]byte{0x0F, 0x8F})
					relocs = append(relocs, Reloc{
						Section: ".text",
						Offset:  uint64(codeBuf.Len()),
						Size:    4,
						Name:    labelOp.Name,
						Addend:  0,
						Kind:    "rel32",
					})
					codeBuf.Write(make([]byte, 4))

				case "jl":
					if len(n.Operands) != 1 {
						return fmt.Errorf("jl needs 1 operand")
					}
					labelOp, ok := n.Operands[0].(LabelOperand)
					if !ok {
						return fmt.Errorf("jl operand must be label: %T", n.Operands[0])
					}
					codeBuf.Write([]byte{0x0F, 0x8C})
					relocs = append(relocs, Reloc{
						Section: ".text",
						Offset:  uint64(codeBuf.Len()),
						Size:    4,
						Name:    labelOp.Name,
						Addend:  0,
						Kind:    "rel32",
					})
					codeBuf.Write(make([]byte, 4))

				case "jge":

					if len(n.Operands) != 1 {
						return fmt.Errorf("jge needs 1 operand")
					}
					labelOp, ok := n.Operands[0].(LabelOperand)
					if !ok {
						return fmt.Errorf("jge operand must be label: %T", n.Operands[0])
					}
					codeBuf.Write([]byte{0x0F, 0x8D})
					relocs = append(relocs, Reloc{
						Section: ".text",
						Offset:  uint64(codeBuf.Len()),
						Size:    4,
						Name:    labelOp.Name,
						Addend:  0,
						Kind:    "rel32",
					})
					codeBuf.Write(make([]byte, 4))

				case "jle":

					if len(n.Operands) != 1 {
						return fmt.Errorf("jle needs 1 operand")
					}
					labelOp, ok := n.Operands[0].(LabelOperand)
					if !ok {
						return fmt.Errorf("jle operand must be label: %T", n.Operands[0])
					}
					codeBuf.Write([]byte{0x0F, 0x8E})
					relocs = append(relocs, Reloc{
						Section: ".text",
						Offset:  uint64(codeBuf.Len()),
						Size:    4,
						Name:    labelOp.Name,
						Addend:  0,
						Kind:    "rel32",
					})
					codeBuf.Write(make([]byte, 4))

				default:
					return fmt.Errorf("unsupported instruction mnemonic: %s", n.Mnemonic)
				}
			}
		}
	}

	code := codeBuf.Bytes()
	data := dataBuf.Bytes()
	codeLen := uint64(len(code))
	dataLen := uint64(len(data))

	textVaddr := baseVaddr + textFileOff
	dataVaddr := textVaddr + codeLen

	for _, r := range relocs {

		var targetAddr uint64
		sym, ok := syms[r.Name]
		if !ok {
			return fmt.Errorf("undefined symbol: %s", r.Name)
		}
		if sym.Section == ".text" {
			targetAddr = textVaddr + sym.Offset
		} else {
			targetAddr = dataVaddr + sym.Offset
		}

		targetAddrWithAdd := uint64(int64(targetAddr) + r.Addend)

		if r.Section == ".text" {
			if r.Offset+uint64(r.Size) > uint64(len(code)) {
				return fmt.Errorf("reloc out of range: %v", r)
			}

			switch r.Kind {
			case "abs64":
				binary.LittleEndian.PutUint64(code[r.Offset:r.Offset+8], targetAddrWithAdd)
			case "abs32":
				binary.LittleEndian.PutUint32(code[r.Offset:r.Offset+4], uint32(targetAddrWithAdd))
			case "rel32":
				rel := int64(targetAddr) - int64(textVaddr) - int64(r.Offset) - 4 + r.Addend
				binary.LittleEndian.PutUint32(code[r.Offset:r.Offset+4], uint32(int32(rel)))
			}
		} else {
			if r.Offset+uint64(r.Size) > uint64(len(data)) {
				return fmt.Errorf("reloc out of range data: %v", r)
			}

			if r.Kind == "abs64" || r.Size == 8 {
				binary.LittleEndian.PutUint64(data[r.Offset:r.Offset+8], targetAddrWithAdd)
			} else if r.Kind == "abs32" || r.Size == 4 {
				binary.LittleEndian.PutUint32(data[r.Offset:r.Offset+4], uint32(targetAddrWithAdd))
			}
		}
	}

	elfBytes, err := elf.BuildElf(codeLen, dataLen, textFileOff, baseVaddr, textVaddr)
	if err != nil {
		return err
	}

	if uint64(len(elfBytes)) < textFileOff+codeLen+dataLen {
		return fmt.Errorf("elf buffer too small: have %d need %d", len(elfBytes), textFileOff+codeLen+dataLen)
	}

	copy(elfBytes[textFileOff:textFileOff+codeLen], code)
	copy(elfBytes[textFileOff+codeLen:textFileOff+codeLen+dataLen], data)

	fout, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fout.Close()
	if _, err := fout.Write(elfBytes); err != nil {
		return err
	}
	if err := fout.Chmod(0755); err != nil {
	}
	return nil
}

func evalToInt64(x interface{}) (int64, error) {
	switch v := x.(type) {
	case NumberExpr:
		return v.Val, nil
	case UnaryExpr:
		n, err := evalToInt64(v.X)
		if err != nil {
			return 0, err
		}
		if v.Op == "-" {
			return -n, nil
		}
		return 0, fmt.Errorf("unsupported unary op: %s", v.Op)
	case BinaryExpr:
		L, err := evalToInt64(v.Left)
		if err != nil {
			return 0, err
		}
		R, err := evalToInt64(v.Right)
		if err != nil {
			return 0, err
		}
		switch v.Op {
		case "+":
			return L + R, nil
		case "-":
			return L - R, nil
		case "*":
			return L * R, nil
		case "/":
			if R == 0 {
				return 0, fmt.Errorf("div by zero")
			}
			return L / R, nil
		default:
			return 0, fmt.Errorf("unsupported binary op: %s", v.Op)
		}
	default:
		return 0, fmt.Errorf("cannot evaluate expr type: %T", x)
	}
}
