package format

type Format int

const (
	FormatUnknown Format = iota
	FormatELF
	FormatPE
	FormatMachO
	FormatRaw
)

func (f Format) String() string {
	switch f {
	case FormatELF:
		return "elf"
	case FormatPE:
		return "pe"
	case FormatMachO:
		return "macho"
	case FormatRaw:
		return "raw"
	default:
		return "unknown"
	}
}

func ParseFormat(s string) Format {
	switch s {
	case "elf":
		return FormatELF
	case "pe", "exe", "dll":
		return FormatPE
	case "macho", "mach", "dylib":
		return FormatMachO
	case "raw", "bin":
		return FormatRaw
	default:
		return FormatUnknown
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
	Kind    int
}

type Section struct {
	Name string
	Data []byte
}

type BuilderInput struct {
	Sections []Section
	Symbols  []Symbol
	Relocs   []Reloc
	Arch     int
	WordSize int
	Entry    string
}

type Builder interface {
	Format() Format
	Build(input *BuilderInput) ([]byte, error)
	Extension() string
}
