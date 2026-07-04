section .text
    global _start

_start:
    mov rax, 6
    mov rbx, 3
    add rax, rbx
    sub rax, 2
    and rax, 15
    or  rax, 1
    imul rax, rbx
    test rax, 1
    jz done

    neg rax
    not rax

done:
    mov rax, 60
    xor rdi, rdi
    syscall

