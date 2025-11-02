# gasm

**gasm** is a modular, nasm-style assembler written entirely in Go. Designed for extensibility and readability, it features a pure handwritten Lexer, Parser, and AST system (no regular expressions) and a comprehensive debug output for AST inspection.

## Features
- **NASM-inspired syntax:** Focused initially on x86_64, easily extendable to arm64/others.
- **Pure-Go Implementation:** No external assembler tools or regex engines; all components are handcrafted for clarity and extensibility.
- **Modular Architecture:** Each architecture (x86_64, arm64, etc.) has its own independent Lexer/Parser/Codegen modules.
- **Powerful AST Debugging:** Includes helpers for visualizing the full parsed structure of your assembly input.
- **Ready for future enhancements:** The codebase is organized for easy addition of more instruction sets, pseudo-ops, macros, and more.

## Project Layout
```
gasm/
├── arch/
│   ├── x86_64/   # Handwritten lexer/parser/ast/debug for x86_64 assembly
│   └── ...       # Future architectures (e.g., arm64)
├── examples/     # Sample asm files
gasm/main.go      # Command-line driver for parsing/testing AST
```

## Usage (Current prototype)
1. Place your nasm-style .asm file in `examples/` (see `examples/test.asm` for syntax examples)
2. Run the main program:
   ```sh
   go run main.go
   ```
   This will lex, parse, and print the AST for your input.

## Design Notes
- The entire toolchain is Go-native. All parsing logic is done manually (no regex!), simulating a C-style lexer/parser machinery for maximal flexibility and explicitness.
- The structure is designed for maintainability: Adding new instructions, features, or architectures is straightforward.
- Parser and AST construction will become even more robust in future updates, including perfect newline and syntax fidelity and error reporting.

## Roadmap / Future Work
- Improved parsing for all nuanced NASM syntax and newlines.
- Bincode/codegen phase for outputting actual machine code.
- Expanding support for additional instruction sets, pseudo-ops, and macro-processing.
- Multiple arch support (arm64, riscv, etc.) and platform-specific features.

---
**gasm** aims to be a practical, hackable, and extensible assembler frontend, providing clarity for language tools enthusiasts and system programmers.
