package elf

import (
	"encoding/binary"
	"gasm/internal/arch"
	"gasm/internal/format"
)

type Builder struct {
	arch arch.Arch
}

func NewBuilder(a arch.Arch) *Builder {
	return &Builder{arch: a}
}

func (b *Builder) Format() format.Format {
	return format.FormatELF
}

func (b *Builder) Extension() string {
	return ""
}

func (b *Builder) Build(input *format.BuilderInput) ([]byte, error) {
	var code, data []byte
	var codeLen, dataLen uint64

	for _, sec := range input.Sections {
		if sec.Name == ".text" {
			code = sec.Data
			codeLen = uint64(len(code))
		} else if sec.Name == ".data" || sec.Name == ".rodata" {
			data = append(data, sec.Data...)
		}
	}
	dataLen = uint64(len(data))

	return BuildELF(codeLen, dataLen, input.WordSize, input.Arch, input.Symbols, input.Relocs)
}

type ELFHeader struct {
	Ident     [16]byte
	Type      uint16
	Machine   uint16
	Version   uint32
	Entry     uint64
	PhOff     uint64
	ShOff     uint64
	Flags     uint32
	EhSize    uint16
	PhEntSize uint16
	PhNum     uint16
	ShEntSize uint16
	ShNum     uint16
	ShStrNdx  uint16
}

type ProgramHeader struct {
	Type   uint32
	Flags  uint32
	Offset uint64
	VAddr  uint64
	PAddr  uint64
	Filesz uint64
	Memsz  uint64
	Align  uint64
}

func BuildELF(codeLen, dataLen uint64, wordSize int, archID int, symbols []format.Symbol, relocs []format.Reloc) ([]byte, error) {
	const pageSize = uint64(0x1000)
	const baseVaddr = uint64(0x400000)

	ehSize := uint64(64)
	phSize := uint64(56)
	textFileOff := pageSize

	payloadSize := codeLen + dataLen
	fileSize := textFileOff + payloadSize

	buf := make([]byte, fileSize)

	buf[0] = 0x7f
	copy(buf[1:], []byte("ELF"))

	if wordSize == 8 {
		buf[4] = 2
	} else {
		buf[4] = 1
	}
	buf[5] = 1
	buf[6] = 1

	binary.LittleEndian.PutUint16(buf[16:], 2)
	binary.LittleEndian.PutUint16(buf[18:], machineFromArch(archID))
	binary.LittleEndian.PutUint32(buf[20:], 1)

	entry := baseVaddr + textFileOff
	binary.LittleEndian.PutUint64(buf[24:], entry)
	binary.LittleEndian.PutUint64(buf[32:], ehSize)
	binary.LittleEndian.PutUint64(buf[40:], 0)
	binary.LittleEndian.PutUint32(buf[48:], 0)
	binary.LittleEndian.PutUint16(buf[52:], uint16(ehSize))
	binary.LittleEndian.PutUint16(buf[54:], uint16(phSize))
	binary.LittleEndian.PutUint16(buf[56:], 1)
	binary.LittleEndian.PutUint16(buf[58:], 0)
	binary.LittleEndian.PutUint16(buf[60:], 0)
	binary.LittleEndian.PutUint16(buf[62:], 0)

	phoff := ehSize
	binary.LittleEndian.PutUint32(buf[phoff+0:], 1)
	binary.LittleEndian.PutUint32(buf[phoff+4:], 7)
	binary.LittleEndian.PutUint64(buf[phoff+8:], textFileOff)
	binary.LittleEndian.PutUint64(buf[phoff+16:], baseVaddr+textFileOff)
	binary.LittleEndian.PutUint64(buf[phoff+24:], baseVaddr+textFileOff)
	binary.LittleEndian.PutUint64(buf[phoff+32:], payloadSize)
	binary.LittleEndian.PutUint64(buf[phoff+40:], payloadSize)
	binary.LittleEndian.PutUint64(buf[phoff+48:], pageSize)

	return buf, nil
}

func machineFromArch(archID int) uint16 {
	switch arch.Arch(archID) {
	case arch.ArchX86:
		return 3
	case arch.ArchX86_64:
		return 0x3E
	case arch.ArchARM:
		return 40
	case arch.ArchARM64:
		return 183
	default:
		return 0x3E
	}
}

func WriteRelocations(buf []byte, relocs []format.Reloc, symbols []format.Symbol, textVaddr, dataVaddr uint64) error {
	for _, r := range relocs {
		var targetAddr uint64
		for _, s := range symbols {
			if s.Name == r.Name {
				if s.Section == ".text" {
					targetAddr = textVaddr + s.Offset
				} else {
					targetAddr = dataVaddr + s.Offset
				}
				break
			}
		}

		targetAddrWithAdd := uint64(int64(targetAddr) + r.Addend)

		if r.Section == ".text" {
			if r.Offset+uint64(r.Size) > uint64(len(buf)) {
				continue
			}

			switch r.Kind {
			case int(arch.RelocAbs64):
				binary.LittleEndian.PutUint64(buf[r.Offset:r.Offset+8], targetAddrWithAdd)
			case int(arch.RelocAbs32):
				binary.LittleEndian.PutUint32(buf[r.Offset:r.Offset+4], uint32(targetAddrWithAdd))
			case int(arch.RelocRel32):
				rel := int64(targetAddr) - int64(textVaddr) - int64(r.Offset) - 4 + r.Addend
				binary.LittleEndian.PutUint32(buf[r.Offset:r.Offset+4], uint32(int32(rel)))
			}
		}
	}
	return nil
}
