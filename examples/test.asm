section .data
    msg db "Hello, world!", 10

section .text
    global _start

_start:
    mov     rax, 1                 ; syscall: write
    mov     rdi, 1                 ; fd = stdout
    mov     rsi, msg               ; buf = msg
    mov     rdx, 13                ; count = len
    syscall

    mov     rax, 60                ; syscall: exit
    xor     rdi, rdi               ; status = 0
    syscall
