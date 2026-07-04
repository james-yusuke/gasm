section .text
    global _start

_start:
    mov rbp, 4096
    mov [rbp-8], 42
    mov rax, [rbp-8]
    lea rsi, [rbp-8]

    mov rax, 60
    xor rdi, rdi
    syscall
