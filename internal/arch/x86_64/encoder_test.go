package x86_64

import (
	"bytes"
	"testing"

	"gasm/internal/ast"
)

func TestEncodeAdditionalInstructions(t *testing.T) {
	enc := NewEncoder()

	tests := []struct {
		name string
		ins  *ast.Instruction
		want []byte
	}{
		{
			name: "and register immediate",
			ins: &ast.Instruction{
				Mnemonic: "and",
				Operands: []ast.Operand{
					ast.RegOperand{Name: "rax"},
					ast.ImmOperand{Val: ast.NumberExpr{Val: 15}},
				},
			},
			want: []byte{0x48, 0x83, 0xE0, 0x0F},
		},
		{
			name: "or registers",
			ins: &ast.Instruction{
				Mnemonic: "or",
				Operands: []ast.Operand{
					ast.RegOperand{Name: "rax"},
					ast.RegOperand{Name: "rbx"},
				},
			},
			want: []byte{0x48, 0x0B, 0xC3},
		},
		{
			name: "imul registers",
			ins: &ast.Instruction{
				Mnemonic: "imul",
				Operands: []ast.Operand{
					ast.RegOperand{Name: "rax"},
					ast.RegOperand{Name: "rbx"},
				},
			},
			want: []byte{0x48, 0x0F, 0xAF, 0xC3},
		},
		{
			name: "test register immediate",
			ins: &ast.Instruction{
				Mnemonic: "test",
				Operands: []ast.Operand{
					ast.RegOperand{Name: "rax"},
					ast.ImmOperand{Val: ast.NumberExpr{Val: 1}},
				},
			},
			want: []byte{0x48, 0xF7, 0xC0, 0x01, 0x00, 0x00, 0x00},
		},
		{
			name: "lea register memory",
			ins: &ast.Instruction{
				Mnemonic: "lea",
				Operands: []ast.Operand{
					ast.RegOperand{Name: "rax"},
					ast.MemOperand{Base: "rbp", Disp: ast.NumberExpr{Val: -8}},
				},
			},
			want: []byte{0x48, 0x8D, 0x45, 0xF8},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := enc.EncodeInstruction(tt.ins)
			if err != nil {
				t.Fatalf("EncodeInstruction returned error: %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("EncodeInstruction bytes = % X, want % X", got, tt.want)
			}
		})
	}
}
