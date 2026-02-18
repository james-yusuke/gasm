package pe

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
	return format.FormatPE
}

func (b *Builder) Extension() string {
	return ".exe"
}

func (b *Builder) Build(input *format.BuilderInput) ([]byte, error) {
	var code, data []byte

	for _, sec := range input.Sections {
		if sec.Name == ".text" {
			code = sec.Data
		} else if sec.Name == ".data" || sec.Name == ".rdata" {
			data = append(data, sec.Data...)
		}
	}

	return BuildPE(code, data, input.WordSize, input.Arch)
}

func BuildPE(code, data []byte, wordSize, archID int) ([]byte, error) {
	is64bit := wordSize == 8

	dosHeader := make([]byte, 64)
	dosHeader[0] = 'M'
	dosHeader[1] = 'Z'
	binary.LittleEndian.PutUint32(dosHeader[60:], uint32(64))

	peSignatureOffset := 64
	peSig := []byte{'P', 'E', 0, 0}

	var fileSize int
	var textOffset, dataOffset uint32

	if is64bit {
		fileSize = 64 + 4 + 240 + 40 + 40
		textOffset = uint32(fileSize)
		fileSize += len(code)
		fileSize = align(fileSize, 512)
		dataOffset = uint32(fileSize)
		fileSize += len(data)
		fileSize = align(fileSize, 512)
	} else {
		fileSize = 64 + 4 + 224 + 40 + 40
		textOffset = uint32(fileSize)
		fileSize += len(code)
		fileSize = align(fileSize, 512)
		dataOffset = uint32(fileSize)
		fileSize += len(data)
		fileSize = align(fileSize, 512)
	}

	buf := make([]byte, fileSize)
	copy(buf, dosHeader)
	copy(buf[peSignatureOffset:], peSig)

	if is64bit {
		coffOffset := peSignatureOffset + 4
		writeCOFFHeader64(buf[coffOffset:], archID)
		peOptOffset := coffOffset + 24
		writePEOptHeader64(buf[peOptOffset:], textOffset, uint32(len(code)))
		secHeaderOffset := peOptOffset + 240
		writeSectionHeaders(buf[secHeaderOffset:], textOffset, dataOffset, uint32(len(code)), uint32(len(data)), is64bit)
	} else {
		coffOffset := peSignatureOffset + 4
		writeCOFFHeader32(buf[coffOffset:], archID)
		peOptOffset := coffOffset + 20
		writePEOptHeader32(buf[peOptOffset:], textOffset, uint32(len(code)))
		secHeaderOffset := peOptOffset + 224
		writeSectionHeaders(buf[secHeaderOffset:], textOffset, dataOffset, uint32(len(code)), uint32(len(data)), is64bit)
	}

	copy(buf[textOffset:], code)
	copy(buf[dataOffset:], data)

	return buf, nil
}

func writeCOFFHeader64(buf []byte, archID int) {
	machine := uint16(0x8664)
	if arch.Arch(archID) == arch.ArchX86 {
		machine = 0x014c
	}
	binary.LittleEndian.PutUint16(buf[0:], machine)
	binary.LittleEndian.PutUint16(buf[2:], 2)
	binary.LittleEndian.PutUint32(buf[4:], 0)
	binary.LittleEndian.PutUint32(buf[8:], 0)
	binary.LittleEndian.PutUint32(buf[12:], 0)
	binary.LittleEndian.PutUint16(buf[16:], 240)
	binary.LittleEndian.PutUint16(buf[18:], 0x22)
	binary.LittleEndian.PutUint16(buf[20:], 0)
}

func writeCOFFHeader32(buf []byte, archID int) {
	binary.LittleEndian.PutUint16(buf[0:], 0x014c)
	binary.LittleEndian.PutUint16(buf[2:], 2)
	binary.LittleEndian.PutUint32(buf[4:], 0)
	binary.LittleEndian.PutUint32(buf[8:], 0)
	binary.LittleEndian.PutUint32(buf[12:], 0)
	binary.LittleEndian.PutUint16(buf[16:], 224)
	binary.LittleEndian.PutUint16(buf[18:], 0x103)
	binary.LittleEndian.PutUint16(buf[20:], 0)
}

func writePEOptHeader64(buf []byte, textOffset, textSize uint32) {
	binary.LittleEndian.PutUint16(buf[0:], 0x20b)
	binary.LittleEndian.PutUint16(buf[2:], 0x17)
	binary.LittleEndian.PutUint32(buf[4:], 0)
	binary.LittleEndian.PutUint32(buf[8:], textSize)
	binary.LittleEndian.PutUint32(buf[12:], 0)
	binary.LittleEndian.PutUint32(buf[16:], 0x1000)
	binary.LittleEndian.PutUint64(buf[24:], 0x10000000+uint64(textOffset))
	binary.LittleEndian.PutUint64(buf[32:], 0x10000000)
	binary.LittleEndian.PutUint64(buf[48:], 0x1000)
	binary.LittleEndian.PutUint64(buf[56:], 0x200000)
	binary.LittleEndian.PutUint64(buf[64:], 0x100000)
	binary.LittleEndian.PutUint32(buf[84:], 6)
	binary.LittleEndian.PutUint32(buf[88:], 0)
	binary.LittleEndian.PutUint64(buf[104:], 0x10000000+uint64(textOffset))
	binary.LittleEndian.PutUint64(buf[112:], 0x10)
	binary.LittleEndian.PutUint64(buf[136:], 0x2000)
	binary.LittleEndian.PutUint64(buf[144:], 0)
	binary.LittleEndian.PutUint16(buf[232:], 2)
}

func writePEOptHeader32(buf []byte, textOffset, textSize uint32) {
	binary.LittleEndian.PutUint16(buf[0:], 0x10b)
	binary.LittleEndian.PutUint16(buf[2:], 0x0e)
	binary.LittleEndian.PutUint32(buf[4:], 0)
	binary.LittleEndian.PutUint32(buf[8:], textSize)
	binary.LittleEndian.PutUint32(buf[12:], 0)
	binary.LittleEndian.PutUint32(buf[16:], 0x1000)
	binary.LittleEndian.PutUint32(buf[24:], 0x10000000+textOffset)
	binary.LittleEndian.PutUint32(buf[28:], 0x10000000)
	binary.LittleEndian.PutUint32(buf[40:], 0x1000)
	binary.LittleEndian.PutUint32(buf[44:], 0x200000)
	binary.LittleEndian.PutUint32(buf[48:], 0x100000)
	binary.LittleEndian.PutUint32(buf[64:], 6)
	binary.LittleEndian.PutUint32(buf[68:], 0)
	binary.LittleEndian.PutUint32(buf[80:], 0x10000000+textOffset)
	binary.LittleEndian.PutUint32(buf[84:], 0x10)
	binary.LittleEndian.PutUint32(buf[104:], 0x2000)
	binary.LittleEndian.PutUint32(buf[108:], 0)
	binary.LittleEndian.PutUint16(buf[208:], 2)
}

func writeSectionHeaders(buf []byte, textOffset, dataOffset, textSize, dataSize uint32, is64bit bool) {
	writeSectionHeader(buf[0:], ".text\000\000\000", textSize, textOffset, 0x60000020)
	if dataSize > 0 {
		writeSectionHeader(buf[40:], ".data\000\000\000", dataSize, dataOffset, 0xC0000040)
	}
}

func writeSectionHeader(buf []byte, name string, size, offset uint32, flags uint32) {
	copy(buf[0:], name)
	binary.LittleEndian.PutUint32(buf[8:], size)
	binary.LittleEndian.PutUint32(buf[12:], 0)
	binary.LittleEndian.PutUint32(buf[16:], size)
	binary.LittleEndian.PutUint32(buf[20:], offset)
	binary.LittleEndian.PutUint32(buf[24:], 0)
	binary.LittleEndian.PutUint32(buf[28:], 0)
	binary.LittleEndian.PutUint32(buf[32:], 0)
	binary.LittleEndian.PutUint32(buf[36:], flags)
}

func align(n, alignment int) int {
	return (n + alignment - 1) & ^(alignment - 1)
}
