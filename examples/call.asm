section .text
    global _start

_start:
    call set_status
    mov rax, 60
    syscall

set_status:
    mov rdi, 0
    ret

