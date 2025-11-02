package elf

import "encoding/binary"

func BuildElf(codeLen, dataLen, textFileOff, baseVaddr, entry uint64) ([]byte, error) {
	const pageSize = uint64(0x1000)
	payloadSize := codeLen + dataLen
	fileSize := textFileOff + payloadSize
	if fileSize > pageSize {
		fileSize = pageSize
	}
	buf := make([]byte, fileSize)

	// e_ident
	buf[0] = 0x7f
	copy(buf[1:], []byte("ELF"))
	buf[4] = 2 // ELFCLASS64
	buf[5] = 1 // ELFDATA2LSB
	buf[6] = 1 // EV_CURRENT

	// ELF header fields
	binary.LittleEndian.PutUint16(buf[16:], 2)     // e_type = ET_EXEC
	binary.LittleEndian.PutUint16(buf[18:], 0x3E)  // e_machine = EM_X86_64
	binary.LittleEndian.PutUint32(buf[20:], 1)     // e_version
	binary.LittleEndian.PutUint64(buf[24:], entry) // e_entry
	binary.LittleEndian.PutUint64(buf[32:], 64)    // e_phoff = 64
	binary.LittleEndian.PutUint64(buf[40:], 0)     // e_shoff = 0 (no sections)
	binary.LittleEndian.PutUint32(buf[48:], 0)     // e_flags
	binary.LittleEndian.PutUint16(buf[52:], 64)    // e_ehsize
	binary.LittleEndian.PutUint16(buf[54:], 56)    // e_phentsize
	binary.LittleEndian.PutUint16(buf[56:], 1)     // e_phnum = 1
	binary.LittleEndian.PutUint16(buf[58:], 0)
	binary.LittleEndian.PutUint16(buf[60:], 0)
	binary.LittleEndian.PutUint16(buf[62:], 0)

	phoff := uint64(64)

	binary.LittleEndian.PutUint32(buf[phoff+0:], 1)                      // p_type = PT_LOAD
	binary.LittleEndian.PutUint32(buf[phoff+4:], 7)                      // p_flags = PF_R|PF_W|PF_X (simple for test)
	binary.LittleEndian.PutUint64(buf[phoff+8:], textFileOff)            // p_offset (file)
	binary.LittleEndian.PutUint64(buf[phoff+16:], baseVaddr+textFileOff) // p_vaddr
	binary.LittleEndian.PutUint64(buf[phoff+24:], baseVaddr+textFileOff) // p_paddr
	binary.LittleEndian.PutUint64(buf[phoff+32:], payloadSize)           // p_filesz
	binary.LittleEndian.PutUint64(buf[phoff+40:], payloadSize)           // p_memsz
	binary.LittleEndian.PutUint64(buf[phoff+48:], pageSize)              // p_align

	return buf, nil
}
