section .data
    msg db "Hello, world!"

section .text
    global _start

_start:
    mov rcx, 3

loop_start:
    mov rax, 1              ; syscall: write
    mov rdi, 1              ; fd = stdout
    mov rsi, msg            ; buf = msg
    mov rdx, 14             ; count = len
    syscall

    dec rcx
    jne loop_start          ; rcx != 0

    mov rax, 60
    xor rdi, rdi
    syscall
