package x86_64

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gasm/internal/arch"
	"gasm/internal/ast"
	"strings"
)

type Encoder struct {
	*arch.BaseEncoder
}

func NewEncoder() *Encoder {
	regs := map[string]int{
		"rax": 0, "eax": 0, "ax": 0, "al": 0, "ah": 0,
		"rcx": 1, "ecx": 1, "cx": 1, "cl": 1, "ch": 1,
		"rdx": 2, "edx": 2, "dx": 2, "dl": 2, "dh": 2,
		"rbx": 3, "ebx": 3, "bx": 3, "bl": 3, "bh": 3,
		"rsp": 4, "esp": 4, "sp": 4, "spl": 4,
		"rbp": 5, "ebp": 5, "bp": 5, "bpl": 5,
		"rsi": 6, "esi": 6, "si": 6, "sil": 6,
		"rdi": 7, "edi": 7, "di": 7, "dil": 7,
		"r8": 8, "r8d": 8, "r8w": 8, "r8b": 8,
		"r9": 9, "r9d": 9, "r9w": 9, "r9b": 9,
		"r10": 10, "r10d": 10, "r10w": 10, "r10b": 10,
		"r11": 11, "r11d": 11, "r11w": 11, "r11b": 11,
		"r12": 12, "r12d": 12, "r12w": 12, "r12b": 12,
		"r13": 13, "r13d": 13, "r13w": 13, "r13b": 13,
		"r14": 14, "r14d": 14, "r14w": 14, "r14b": 14,
		"r15": 15, "r15d": 15, "r15w": 15, "r15b": 15,
	}
	return &Encoder{BaseEncoder: arch.NewBaseEncoder(arch.ArchX86_64, 8, regs)}
}

func (e *Encoder) EncodeInstruction(ins *ast.Instruction) ([]byte, error) {
	var buf bytes.Buffer
	mn := strings.ToLower(ins.Mnemonic)

	switch mn {
	case "mov":
		return e.encodeMov(&buf, ins)
	case "xor":
		return e.encodeXor(&buf, ins)
	case "add":
		return e.encodeAdd(&buf, ins)
	case "sub":
		return e.encodeSub(&buf, ins)
	case "inc":
		return e.encodeInc(&buf, ins)
	case "dec":
		return e.encodeDec(&buf, ins)
	case "cmp":
		return e.encodeCmp(&buf, ins)
	case "jmp":
		return e.encodeJmp(&buf, ins)
	case "je", "jz":
		return e.encodeJcc(&buf, ins, 0x84)
	case "jne", "jnz":
		return e.encodeJcc(&buf, ins, 0x85)
	case "jg":
		return e.encodeJcc(&buf, ins, 0x8F)
	case "jl":
		return e.encodeJcc(&buf, ins, 0x8C)
	case "jge":
		return e.encodeJcc(&buf, ins, 0x8D)
	case "jle":
		return e.encodeJcc(&buf, ins, 0x8E)
	case "ja":
		return e.encodeJcc(&buf, ins, 0x87)
	case "jb":
		return e.encodeJcc(&buf, ins, 0x82)
	case "call":
		return e.encodeCall(&buf, ins)
	case "ret":
		buf.WriteByte(0xC3)
		return buf.Bytes(), nil
	case "push":
		return e.encodePush(&buf, ins)
	case "pop":
		return e.encodePop(&buf, ins)
	case "syscall":
		buf.Write([]byte{0x0F, 0x05})
		return buf.Bytes(), nil
	case "nop":
		buf.WriteByte(0x90)
		return buf.Bytes(), nil
	case "int":
		return e.encodeInt(&buf, ins)
	case "lea":
		return e.encodeLea(&buf, ins)
	case "test":
		return e.encodeTest(&buf, ins)
	default:
		return nil, fmt.Errorf("unsupported instruction: %s", ins.Mnemonic)
	}
}

func (e *Encoder) encodeMov(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	if len(ins.Operands) != 2 {
		return nil, fmt.Errorf("mov requires 2 operands")
	}

	dst := ins.Operands[0]
	src := ins.Operands[1]

	if rd, ok := dst.(ast.RegOperand); ok {
		regID, ok := e.Registers()[strings.ToLower(rd.Name)]
		if !ok {
			return nil, fmt.Errorf("unknown register: %s", rd.Name)
		}

		switch s := src.(type) {
		case ast.ImmOperand:
			return e.encodeMovRegImm(buf, regID, s.Val, rd.Name)
		case ast.RegOperand:
			srcID, ok := e.Registers()[strings.ToLower(s.Name)]
			if !ok {
				return nil, fmt.Errorf("unknown register: %s", s.Name)
			}
			return e.encodeMovRegReg(buf, regID, srcID, rd.Name, s.Name)
		case ast.MemOperand:
			return e.encodeMovRegMem(buf, regID, s, rd.Name)
		case ast.LabelOperand:
			e.writeRexIfNeeded(buf, regID)
			buf.WriteByte(byte(0xB8 | (regID & 7)))
			buf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
			return buf.Bytes(), nil
		default:
			return nil, fmt.Errorf("unsupported mov src: %T", src)
		}
	}

	if md, ok := dst.(ast.MemOperand); ok {
		if rs, ok := src.(ast.RegOperand); ok {
			regID, ok := e.Registers()[strings.ToLower(rs.Name)]
			if !ok {
				return nil, fmt.Errorf("unknown register: %s", rs.Name)
			}
			return e.encodeMovMemReg(buf, md, regID, rs.Name)
		}
		if imm, ok := src.(ast.ImmOperand); ok {
			return e.encodeMovMemImm(buf, md, imm.Val)
		}
	}

	return nil, fmt.Errorf("unsupported mov operands: %T <- %T", dst, src)
}

func (e *Encoder) encodeMovRegImm(buf *bytes.Buffer, regID int, val ast.Expr, regName string) ([]byte, error) {
	if num, ok := val.(ast.NumberExpr); ok {
		e.writeRex(buf, 0, regID, true, regName)
		op := byte(0xB8 | byte(regID&7))
		buf.WriteByte(op)
		var tmp [8]byte
		binary.LittleEndian.PutUint64(tmp[:], uint64(num.Val))
		buf.Write(tmp[:])
		return buf.Bytes(), nil
	}
	return nil, fmt.Errorf("mov reg, imm requires NumberExpr for now")
}

func (e *Encoder) encodeMovRegReg(buf *bytes.Buffer, dstID, srcID int, dstName, srcName string) ([]byte, error) {
	e.writeRex(buf, srcID, dstID, true, dstName)
	buf.WriteByte(0x89)
	e.writeModRM(buf, srcID, dstID, 0xC0)
	return buf.Bytes(), nil
}

func (e *Encoder) encodeMovRegMem(buf *bytes.Buffer, regID int, mem ast.MemOperand, regName string) ([]byte, error) {
	e.writeRex(buf, regID, 0, true, regName)
	buf.WriteByte(0x8B)
	e.writeModRM(buf, regID, 0, 0x00)
	return buf.Bytes(), nil
}

func (e *Encoder) encodeMovMemReg(buf *bytes.Buffer, mem ast.MemOperand, regID int, regName string) ([]byte, error) {
	e.writeRex(buf, regID, 0, true, regName)
	buf.WriteByte(0x89)
	e.writeModRM(buf, regID, 0, 0x00)
	return buf.Bytes(), nil
}

func (e *Encoder) encodeMovMemImm(buf *bytes.Buffer, mem ast.MemOperand, val ast.Expr) ([]byte, error) {
	return nil, fmt.Errorf("mov mem, imm not yet implemented")
}

func (e *Encoder) encodeXor(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	if len(ins.Operands) != 2 {
		return nil, fmt.Errorf("xor requires 2 operands")
	}

	dst, ok := ins.Operands[0].(ast.RegOperand)
	if !ok {
		return nil, fmt.Errorf("xor dst must be register")
	}
	src, ok := ins.Operands[1].(ast.RegOperand)
	if !ok {
		return nil, fmt.Errorf("xor src must be register")
	}

	dstID, ok := e.Registers()[strings.ToLower(dst.Name)]
	if !ok {
		return nil, fmt.Errorf("unknown register: %s", dst.Name)
	}
	srcID, ok := e.Registers()[strings.ToLower(src.Name)]
	if !ok {
		return nil, fmt.Errorf("unknown register: %s", src.Name)
	}

	e.writeRexIfNeeded(buf, dstID, srcID)
	buf.WriteByte(0x31)
	e.writeModRM(buf, srcID, dstID, 0xC0)
	return buf.Bytes(), nil
}

func (e *Encoder) encodeAdd(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	return e.encodeArithRR(buf, ins, 0x03, 0x01, 0x81, 0x83, 0)
}

func (e *Encoder) encodeSub(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	return e.encodeArithRR(buf, ins, 0x2B, 0x29, 0x81, 0x83, 5)
}

func (e *Encoder) encodeCmp(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	return e.encodeArithRR(buf, ins, 0x3B, 0x39, 0x81, 0x83, 7)
}

func (e *Encoder) encodeArithRR(buf *bytes.Buffer, ins *ast.Instruction, opRegReg, opRegRegAlt, opImm32, opImm8, extField int) ([]byte, error) {
	if len(ins.Operands) != 2 {
		return nil, fmt.Errorf("%s requires 2 operands", ins.Mnemonic)
	}

	dst, ok := ins.Operands[0].(ast.RegOperand)
	if !ok {
		return nil, fmt.Errorf("%s dst must be register", ins.Mnemonic)
	}
	dstID, ok := e.Registers()[strings.ToLower(dst.Name)]
	if !ok {
		return nil, fmt.Errorf("unknown register: %s", dst.Name)
	}

	switch src := ins.Operands[1].(type) {
	case ast.RegOperand:
		srcID, ok := e.Registers()[strings.ToLower(src.Name)]
		if !ok {
			return nil, fmt.Errorf("unknown register: %s", src.Name)
		}
		e.writeRex(buf, dstID, srcID, true, dst.Name)
		buf.WriteByte(byte(opRegReg))
		e.writeModRM(buf, dstID, srcID, 0xC0)
	case ast.ImmOperand:
		if num, ok := src.Val.(ast.NumberExpr); ok {
			if num.Val >= -128 && num.Val <= 127 {
				e.writeRex(buf, extField, dstID, true, dst.Name)
				buf.WriteByte(byte(opImm8))
				e.writeModRM(buf, extField, dstID, 0xC0)
				buf.WriteByte(byte(num.Val))
			} else {
				e.writeRex(buf, extField, dstID, true, dst.Name)
				buf.WriteByte(byte(opImm32))
				e.writeModRM(buf, extField, dstID, 0xC0)
				var tmp [4]byte
				binary.LittleEndian.PutUint32(tmp[:], uint32(num.Val))
				buf.Write(tmp[:])
			}
		} else {
			return nil, fmt.Errorf("%s immediate requires NumberExpr", ins.Mnemonic)
		}
	default:
		return nil, fmt.Errorf("%s unsupported src operand: %T", ins.Mnemonic, src)
	}

	return buf.Bytes(), nil
}

func (e *Encoder) encodeInc(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	if len(ins.Operands) != 1 {
		return nil, fmt.Errorf("inc requires 1 operand")
	}
	rd, ok := ins.Operands[0].(ast.RegOperand)
	if !ok {
		return nil, fmt.Errorf("inc operand must be register")
	}
	regID, ok := e.Registers()[strings.ToLower(rd.Name)]
	if !ok {
		return nil, fmt.Errorf("unknown register: %s", rd.Name)
	}
	e.writeRex(buf, 0, regID, true, rd.Name)
	buf.WriteByte(0xFF)
	e.writeModRM(buf, 0, regID, 0xC0)
	return buf.Bytes(), nil
}

func (e *Encoder) encodeDec(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	if len(ins.Operands) != 1 {
		return nil, fmt.Errorf("dec requires 1 operand")
	}
	rd, ok := ins.Operands[0].(ast.RegOperand)
	if !ok {
		return nil, fmt.Errorf("dec operand must be register")
	}
	regID, ok := e.Registers()[strings.ToLower(rd.Name)]
	if !ok {
		return nil, fmt.Errorf("unknown register: %s", rd.Name)
	}
	e.writeRex(buf, 1, regID, true, rd.Name)
	buf.WriteByte(0xFF)
	e.writeModRM(buf, 1, regID, 0xC0)
	return buf.Bytes(), nil
}

func (e *Encoder) encodeJmp(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	if len(ins.Operands) != 1 {
		return nil, fmt.Errorf("jmp requires 1 operand")
	}
	_, ok := ins.Operands[0].(ast.LabelOperand)
	if !ok {
		return nil, fmt.Errorf("jmp operand must be label")
	}
	buf.WriteByte(0xE9)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00})
	return buf.Bytes(), nil
}

func (e *Encoder) encodeJcc(buf *bytes.Buffer, ins *ast.Instruction, opcode2 byte) ([]byte, error) {
	if len(ins.Operands) != 1 {
		return nil, fmt.Errorf("%s requires 1 operand", ins.Mnemonic)
	}
	_, ok := ins.Operands[0].(ast.LabelOperand)
	if !ok {
		return nil, fmt.Errorf("%s operand must be label", ins.Mnemonic)
	}
	buf.Write([]byte{0x0F, opcode2})
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00})
	return buf.Bytes(), nil
}

func (e *Encoder) encodeCall(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	if len(ins.Operands) != 1 {
		return nil, fmt.Errorf("call requires 1 operand")
	}
	_, ok := ins.Operands[0].(ast.LabelOperand)
	if !ok {
		return nil, fmt.Errorf("call operand must be label")
	}
	buf.WriteByte(0xE8)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00})
	return buf.Bytes(), nil
}

func (e *Encoder) encodePush(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	if len(ins.Operands) != 1 {
		return nil, fmt.Errorf("push requires 1 operand")
	}
	rd, ok := ins.Operands[0].(ast.RegOperand)
	if !ok {
		return nil, fmt.Errorf("push operand must be register")
	}
	regID, ok := e.Registers()[strings.ToLower(rd.Name)]
	if !ok {
		return nil, fmt.Errorf("unknown register: %s", rd.Name)
	}
	if regID >= 8 {
		buf.WriteByte(0x41)
	}
	buf.WriteByte(byte(0x50 | (regID & 7)))
	return buf.Bytes(), nil
}

func (e *Encoder) encodePop(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	if len(ins.Operands) != 1 {
		return nil, fmt.Errorf("pop requires 1 operand")
	}
	rd, ok := ins.Operands[0].(ast.RegOperand)
	if !ok {
		return nil, fmt.Errorf("pop operand must be register")
	}
	regID, ok := e.Registers()[strings.ToLower(rd.Name)]
	if !ok {
		return nil, fmt.Errorf("unknown register: %s", rd.Name)
	}
	if regID >= 8 {
		buf.WriteByte(0x41)
	}
	buf.WriteByte(byte(0x58 | (regID & 7)))
	return buf.Bytes(), nil
}

func (e *Encoder) encodeInt(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	if len(ins.Operands) != 1 {
		return nil, fmt.Errorf("int requires 1 operand")
	}
	imm, ok := ins.Operands[0].(ast.ImmOperand)
	if !ok {
		return nil, fmt.Errorf("int operand must be immediate")
	}
	num, ok := imm.Val.(ast.NumberExpr)
	if !ok {
		return nil, fmt.Errorf("int immediate must be number")
	}
	buf.WriteByte(0xCD)
	buf.WriteByte(byte(num.Val))
	return buf.Bytes(), nil
}

func (e *Encoder) encodeLea(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	return nil, fmt.Errorf("lea not yet implemented")
}

func (e *Encoder) encodeTest(buf *bytes.Buffer, ins *ast.Instruction) ([]byte, error) {
	return nil, fmt.Errorf("test not yet implemented")
}

func (e *Encoder) writeRex(buf *bytes.Buffer, regField, rmField int, needW bool, regName string) {
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
	buf.WriteByte(rex)
}

func (e *Encoder) writeRexIfNeeded(buf *bytes.Buffer, regs ...int) {
	var rex byte = 0x48
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
	buf.WriteByte(rex)
}

func (e *Encoder) writeModRM(buf *bytes.Buffer, regField, rmField int, base byte) {
	modrm := base | byte((regField&7)<<3) | byte(rmField&7)
	buf.WriteByte(modrm)
}
