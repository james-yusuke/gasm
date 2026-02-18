package arch

import "gasm/internal/ast"

type Arch int

const (
	ArchUnknown Arch = iota
	ArchX86
	ArchX86_64
	ArchARM
	ArchARM64
)

func (a Arch) String() string {
	switch a {
	case ArchX86:
		return "x86"
	case ArchX86_64:
		return "x86_64"
	case ArchARM:
		return "arm"
	case ArchARM64:
		return "arm64"
	default:
		return "unknown"
	}
}

func ParseArch(s string) Arch {
	switch s {
	case "x86", "i386", "i686":
		return ArchX86
	case "x86_64", "amd64", "x64":
		return ArchX86_64
	case "arm", "arm32":
		return ArchARM
	case "arm64", "aarch64":
		return ArchARM64
	default:
		return ArchUnknown
	}
}

type Symbol struct {
	Name    string
	Section string
	Offset  uint64
	Size    uint64
	Global  bool
}

type Reloc struct {
	Section string
	Offset  uint64
	Size    int
	Name    string
	Addend  int64
	Kind    RelocKind
}

type RelocKind int

const (
	RelocAbs64 RelocKind = iota
	RelocAbs32
	RelocRel32
	RelocRel64
	RelocCall
	RelocBranch
)

type Section struct {
	Name string
	Data []byte
}

type AssemblyResult struct {
	Sections []Section
	Symbols  []Symbol
	Relocs   []Reloc
}

type Encoder interface {
	Arch() Arch
	WordSize() int
	EncodeInstruction(ins *ast.Instruction) ([]byte, error)
	Registers() map[string]int
	IsRegister(name string) bool
}

type EncoderFunc func(ins *ast.Instruction) ([]byte, error)

type BaseEncoder struct {
	arch      Arch
	wordSize  int
	registers map[string]int
}

func NewBaseEncoder(arch Arch, wordSize int, registers map[string]int) *BaseEncoder {
	return &BaseEncoder{arch: arch, wordSize: wordSize, registers: registers}
}

func (e *BaseEncoder) Arch() Arch                { return e.arch }
func (e *BaseEncoder) WordSize() int             { return e.wordSize }
func (e *BaseEncoder) Registers() map[string]int { return e.registers }
func (e *BaseEncoder) IsRegister(name string) bool {
	_, ok := e.registers[name]
	return ok
}
