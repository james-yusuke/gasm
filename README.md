# gasm

gasm is a small NASM-style assembler written in Go for learning how assemblers work.

This project is intentionally educational. It is useful for reading, experimenting, and extending lexer/parser/AST/encoder code, but it is not a production assembler and should not be used to build trusted release binaries.

## What You Can Learn

- How assembly text is tokenized by a handwritten lexer.
- How a parser turns instructions, labels, directives, and data declarations into an AST.
- How a simple x86_64 encoder maps a subset of instructions to machine code.
- How labels and relocations are collected before building an ELF or PE-style output.
- How a small compiler-like tool can be split into clear internal packages.

## Current Features

- NASM-inspired syntax for a learning-sized x86_64 subset.
- Pure Go implementation with no external assembler dependency.
- Parser support for labels, directives, data declarations, strings, numeric expressions, registers, immediates, and simple memory operands.
- Basic binary output through the internal format builders.
- Examples that are assembled in CI to keep sample code from drifting.

## Supported Assembly Subset

The x86_64 encoder currently supports:

- Data movement: `mov`, `lea`
- Arithmetic and bit operations: `add`, `sub`, `and`, `or`, `xor`, `inc`, `dec`, `neg`, `not`
- Multiplication and division forms: `mul`, `imul`, `div`, `idiv`
- Comparisons and tests: `cmp`, `test`
- Branching: `jmp`, `je`/`jz`, `jne`/`jnz`, `jg`, `jl`, `jge`, `jle`, `ja`, `jb`, `call`, `ret`
- Stack and system instructions: `push`, `pop`, `syscall`, `int`, `nop`

Memory addressing is deliberately simple. Examples such as `[rbp-8]`, `[rax]`, and numeric displacements are good learning targets; complex addressing modes are still future work.

## Project Layout

```text
cmd/gasm/                 CLI entry point
examples/                 Small assembly programs used as smoke tests
internal/ast/             AST node definitions
internal/lexer/           Handwritten lexer
internal/parser/          Parser and expression handling
internal/arch/            Architecture abstractions
internal/arch/x86_64/     x86_64 instruction encoder
internal/asm/             Assembly pipeline and relocation handling
internal/format/          Output format builders
.github/workflows/        Pull request and push checks
```

## Usage

Build the CLI:

```sh
go build ./cmd/gasm
```

Assemble an example:

```sh
go run ./cmd/gasm examples/test.asm hello
```

Run the checks used by CI:

```sh
gofmt -w .
go test ./...
go vet ./...
```

## Examples

- `examples/test.asm`: minimal Linux `write` + `exit` syscall flow.
- `examples/jne.asm`: loop with `dec` and `jne`.
- `examples/arithmetic.asm`: arithmetic, bit operations, `imul`, `test`, and conditional branching.
- `examples/memory.asm`: simple stack-like memory addressing and `lea`.
- `examples/call.asm`: `call` and `ret`.

## Contributing

Small, focused pull requests are easiest to review. Good learning-sized improvements include:

- Adding one instruction form with tests.
- Expanding parser support for one syntax feature.
- Adding an example that demonstrates a real assembler concept.
- Improving error messages for malformed assembly.

Pull requests run GitHub Actions checks for formatting, tests, vet, and CLI build health.

## Safety Note

Assembly input and generated binaries should be treated as unsafe unless you understand and reviewed them. See [SECURITY.md](SECURITY.md) for reporting guidance.
