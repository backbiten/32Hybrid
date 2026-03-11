# 32Hybrid
A reinvention and reinnovation of 32-bit architecture from the 70's and 80's

## Overview

32Hybrid is a clean, well-tested Python implementation of a 32-bit CPU emulator
and assembler, designed to avoid the classic pitfalls and subtle bugs that
plagued real hardware and emulators of the era.

### Key design decisions that prevent common bugs

| Classic 32-bit bug | How 32Hybrid avoids it |
|---|---|
| Ambiguous immediate encoding (flag bit collides with value bits) | `rb == 0xF` in the instruction word is a clean sentinel — the 12-bit immediate occupies its own field without overlap |
| Silent integer overflow | Carry (CF) and overflow (OF) flags are computed independently and correctly for both signed and unsigned arithmetic |
| Wrong sign extension on right-shift | `SHR` is always logical (zero-fills); `SAR` is always arithmetic (sign-extends) — no ambiguity |
| Division by zero corruption | Raises `ZeroDivisionError` before touching any register |
| Runaway programs | `cpu.run(max_steps=N)` hard-limits execution |
| Out-of-bounds memory access | Every access checks bounds and raises `MemoryError` |

## Files

| File | Purpose |
|---|---|
| `cpu32.py` | CPU core — registers, ALU, memory, instruction execution |
| `assembler.py` | Two-pass assembler — source text → list of 32-bit words |
| `tests/test_cpu32.py` | 95 unit, integration and assembler tests |

## Architecture

- **16 general-purpose 32-bit registers**: R0–R14 and R15 (SP, the stack pointer)
- **Separate program counter (PC)**
- **4 condition flags**: ZF (zero), SF (sign), CF (carry), OF (signed overflow)
- **Little-endian 32-bit memory** (64 KiB by default, configurable)
- **Fixed-width 32-bit instruction words**

### Instruction encoding

```
[31:24]  opcode   (8 bits)
[23:20]  Rdst     (4 bits)
[19:16]  Ra       (4 bits)
[15:12]  Rb       (4 bits) — 0xF signals "use sign-extended imm12 instead"
[11: 0]  imm12    (12 bits, sign-extended when used)
```

Some instructions (MOV imm32, branch targets, LOAD/STORE addresses) consume a
second 32-bit word for a full 32-bit operand.

### Instruction set summary

```
NOP                       No operation
HALT                      Stop execution
MOV  Rdst, Rsrc           Copy register
MOV  Rdst, #imm32         Load 32-bit immediate (two-word encoding)
ADD  Rdst, Ra, Rb/#imm    Rdst = Ra + Rb  (sets ZF, SF, CF, OF)
SUB  Rdst, Ra, Rb/#imm    Rdst = Ra - Rb  (sets ZF, SF, CF, OF)
MUL  Rdst, Ra, Rb         Rdst = Ra * Rb (low 32 bits)
DIV  Rdst, Ra, Rb         Rdst = Ra / Rb (unsigned)
AND  Rdst, Ra, Rb/#imm    Rdst = Ra & Rb
OR   Rdst, Ra, Rb/#imm    Rdst = Ra | Rb
XOR  Rdst, Ra, Rb/#imm    Rdst = Ra ^ Rb
NOT  Rdst, Ra             Rdst = ~Ra
SHL  Rdst, Ra, Rb/#imm    Rdst = Ra << shift (logical)
SHR  Rdst, Ra, Rb/#imm    Rdst = Ra >> shift (logical, zero-fill)
SAR  Rdst, Ra, Rb/#imm    Rdst = Ra >> shift (arithmetic, sign-extend)
CMP  Ra, Rb/#imm          Flags from Ra - Rb (no register write)
JMP  label/#addr          Unconditional jump
JEQ  label/#addr          Jump if ZF=1
JNE  label/#addr          Jump if ZF=0
JLT  label/#addr          Jump if SF≠OF  (signed <)
JGT  label/#addr          Jump if ZF=0 and SF=OF  (signed >)
JLE  label/#addr          Jump if ZF=1 or SF≠OF   (signed ≤)
JGE  label/#addr          Jump if SF=OF  (signed ≥)
CALL label/#addr          Push PC, jump
RET                       Pop PC, jump
PUSH Rsrc                 Push register to stack
POP  Rdst                 Pop from stack to register
LOAD  Rdst, [addr32]      Load 32-bit word from memory
STORE [addr32], Rsrc      Store 32-bit word to memory
.word value               Emit raw 32-bit literal
```

## Usage

```python
from assembler import assemble
from cpu32 import CPU

source = """
    MOV R0, #1        ; result
    MOV R1, #10       ; counter
loop:
    MUL R0, R0, R1
    SUB R1, R1, #1
    CMP R1, #0
    JNE loop
    HALT
"""

program = assemble(source)
cpu = CPU()
cpu.load_program(program)
cpu.run()
print(cpu.registers[0])   # 3628800  (10!)
```

## Running the tests

```bash
python -m pytest tests/
```
