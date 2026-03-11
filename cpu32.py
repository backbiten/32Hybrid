"""
32Hybrid CPU Emulator
A clean reinvention of 32-bit architecture from the 70s and 80s,
designed to avoid common pitfalls and bugs of the era.
"""

WORD_SIZE = 32
WORD_MASK = 0xFFFFFFFF
SIGN_BIT = 0x80000000

# Instruction opcodes
OP_NOP   = 0x00  # No operation
OP_LOAD  = 0x01  # LOAD  Rdst, [addr]      - load from memory
OP_STORE = 0x02  # STORE [addr], Rsrc      - store to memory
OP_MOV   = 0x03  # MOV   Rdst, Rsrc/imm    - register copy / load immediate
OP_ADD   = 0x04  # ADD   Rdst, Ra, Rb/imm
OP_SUB   = 0x05  # SUB   Rdst, Ra, Rb/imm
OP_MUL   = 0x06  # MUL   Rdst, Ra, Rb
OP_DIV   = 0x07  # DIV   Rdst, Ra, Rb       - unsigned divide
OP_AND   = 0x08  # AND   Rdst, Ra, Rb/imm
OP_OR    = 0x09  # OR    Rdst, Ra, Rb/imm
OP_XOR   = 0x0A  # XOR   Rdst, Ra, Rb/imm
OP_NOT   = 0x0B  # NOT   Rdst, Ra
OP_SHL   = 0x0C  # SHL   Rdst, Ra, shift
OP_SHR   = 0x0D  # SHR   Rdst, Ra, shift    - logical (unsigned) right shift
OP_SAR   = 0x0E  # SAR   Rdst, Ra, shift    - arithmetic (signed) right shift
OP_CMP   = 0x0F  # CMP   Ra, Rb/imm         - set flags only
OP_JMP   = 0x10  # JMP   addr
OP_JEQ   = 0x11  # JEQ   addr               - jump if equal (ZF=1)
OP_JNE   = 0x12  # JNE   addr               - jump if not equal (ZF=0)
OP_JLT   = 0x13  # JLT   addr               - jump if less than (signed, SF≠OF)
OP_JGT   = 0x14  # JGT   addr               - jump if greater than (signed, ZF=0 and SF=OF)
OP_JLE   = 0x15  # JLE   addr               - jump if less or equal (ZF=1 or SF≠OF)
OP_JGE   = 0x16  # JGE   addr               - jump if greater or equal (signed, SF=OF)
OP_CALL  = 0x17  # CALL  addr               - push PC, jump
OP_RET   = 0x18  # RET                      - pop PC
OP_PUSH  = 0x19  # PUSH  Rsrc
OP_POP   = 0x1A  # POP   Rdst
OP_HALT  = 0x1B  # HALT

NUM_REGS = 16   # R0-R15; R15 is the stack pointer (SP)
SP = 15         # Stack pointer register index
MEM_SIZE = 1 << 16  # 64 KiB default memory


def _to_signed32(value: int) -> int:
    """Interpret a 32-bit unsigned integer as a signed integer."""
    value &= WORD_MASK
    if value & SIGN_BIT:
        return value - (1 << WORD_SIZE)
    return value


def _to_unsigned32(value: int) -> int:
    """Mask an integer to 32 bits (two's-complement unsigned)."""
    return value & WORD_MASK


class Flags:
    """CPU condition flags."""

    def __init__(self) -> None:
        self.zero: bool = False      # ZF - result is zero
        self.sign: bool = False      # SF - result is negative
        self.carry: bool = False     # CF - unsigned overflow / borrow
        self.overflow: bool = False  # OF - signed overflow

    def update(self, result: int, *, carry: bool = False, overflow: bool = False) -> None:
        """Update flags from an ALU result (result should be the raw Python int)."""
        masked = result & WORD_MASK
        self.zero = masked == 0
        self.sign = bool(masked & SIGN_BIT)
        self.carry = carry
        self.overflow = overflow

    def __repr__(self) -> str:
        return (
            f"Flags(ZF={int(self.zero)}, SF={int(self.sign)}, "
            f"CF={int(self.carry)}, OF={int(self.overflow)})"
        )


class CPU:
    """
    32Hybrid CPU emulator.

    Registers R0-R14 are general-purpose 32-bit registers.
    R15 is the stack pointer (SP), initialised to the top of memory.
    The program counter (PC) is a separate register.
    """

    def __init__(self, memory_size: int = MEM_SIZE) -> None:
        self.registers: list[int] = [0] * NUM_REGS
        self.registers[SP] = memory_size - 4  # Stack starts at top of memory
        self.pc: int = 0
        self.flags = Flags()
        self.memory: bytearray = bytearray(memory_size)
        self.halted: bool = False
        self._memory_size = memory_size

    # ------------------------------------------------------------------
    # Memory helpers
    # ------------------------------------------------------------------

    def _check_addr(self, addr: int, size: int = 4) -> None:
        addr = _to_unsigned32(addr)
        if addr + size > self._memory_size:
            raise MemoryError(
                f"Address 0x{addr:08X} out of range (memory size {self._memory_size:#x})"
            )

    def mem_read32(self, addr: int) -> int:
        """Read a 32-bit little-endian word from memory."""
        addr = _to_unsigned32(addr)
        self._check_addr(addr, 4)
        b = self.memory
        return b[addr] | (b[addr + 1] << 8) | (b[addr + 2] << 16) | (b[addr + 3] << 24)

    def mem_write32(self, addr: int, value: int) -> None:
        """Write a 32-bit little-endian word to memory."""
        addr = _to_unsigned32(addr)
        value = _to_unsigned32(value)
        self._check_addr(addr, 4)
        b = self.memory
        b[addr]     = value & 0xFF
        b[addr + 1] = (value >> 8)  & 0xFF
        b[addr + 2] = (value >> 16) & 0xFF
        b[addr + 3] = (value >> 24) & 0xFF

    def mem_read8(self, addr: int) -> int:
        addr = _to_unsigned32(addr)
        self._check_addr(addr, 1)
        return self.memory[addr]

    def mem_write8(self, addr: int, value: int) -> None:
        addr = _to_unsigned32(addr)
        self._check_addr(addr, 1)
        self.memory[addr] = value & 0xFF

    # ------------------------------------------------------------------
    # Stack helpers
    # ------------------------------------------------------------------

    def _push(self, value: int) -> None:
        self.registers[SP] = _to_unsigned32(self.registers[SP] - 4)
        self.mem_write32(self.registers[SP], value)

    def _pop(self) -> int:
        value = self.mem_read32(self.registers[SP])
        self.registers[SP] = _to_unsigned32(self.registers[SP] + 4)
        return value

    # ------------------------------------------------------------------
    # ALU operations (all return unsigned 32-bit results and set flags)
    # ------------------------------------------------------------------

    def _alu_add(self, a: int, b: int) -> int:
        a = _to_unsigned32(a)
        b = _to_unsigned32(b)
        raw = a + b
        result = raw & WORD_MASK
        carry = raw > WORD_MASK
        # Signed overflow: both inputs have same sign but result has different sign
        overflow = bool(
            (~(a ^ b) & (a ^ result)) & SIGN_BIT
        )
        self.flags.update(result, carry=carry, overflow=overflow)
        return result

    def _alu_sub(self, a: int, b: int) -> int:
        a = _to_unsigned32(a)
        b = _to_unsigned32(b)
        raw = a - b
        result = raw & WORD_MASK
        carry = raw < 0  # borrow
        # Signed overflow: inputs have different signs and result sign ≠ a sign
        overflow = bool(
            ((a ^ b) & (a ^ result)) & SIGN_BIT
        )
        self.flags.update(result, carry=carry, overflow=overflow)
        return result

    def _alu_mul(self, a: int, b: int) -> int:
        a = _to_unsigned32(a)
        b = _to_unsigned32(b)
        result = (a * b) & WORD_MASK
        self.flags.update(result, carry=False, overflow=False)
        return result

    def _alu_div(self, a: int, b: int) -> int:
        a = _to_unsigned32(a)
        b = _to_unsigned32(b)
        if b == 0:
            raise ZeroDivisionError("Division by zero")
        result = (a // b) & WORD_MASK
        self.flags.update(result, carry=False, overflow=False)
        return result

    def _alu_and(self, a: int, b: int) -> int:
        result = _to_unsigned32(a) & _to_unsigned32(b)
        self.flags.update(result)
        return result

    def _alu_or(self, a: int, b: int) -> int:
        result = _to_unsigned32(a) | _to_unsigned32(b)
        self.flags.update(result)
        return result

    def _alu_xor(self, a: int, b: int) -> int:
        result = _to_unsigned32(a) ^ _to_unsigned32(b)
        self.flags.update(result)
        return result

    def _alu_not(self, a: int) -> int:
        result = (~_to_unsigned32(a)) & WORD_MASK
        self.flags.update(result)
        return result

    def _alu_shl(self, a: int, shift: int) -> int:
        a = _to_unsigned32(a)
        shift = shift & 31  # Clamp to 0-31 (avoids undefined behaviour)
        if shift == 0:
            result = a
            self.flags.update(result, carry=False, overflow=False)
            return result
        carry = bool((a >> (32 - shift)) & 1)
        result = (a << shift) & WORD_MASK
        overflow = bool(result & SIGN_BIT) != carry  # MSB changed
        self.flags.update(result, carry=carry, overflow=overflow)
        return result

    def _alu_shr(self, a: int, shift: int) -> int:
        """Logical (unsigned) right shift."""
        a = _to_unsigned32(a)
        shift = shift & 31
        if shift == 0:
            result = a
            self.flags.update(result, carry=False, overflow=False)
            return result
        carry = bool((a >> (shift - 1)) & 1)
        result = (a >> shift) & WORD_MASK
        self.flags.update(result, carry=carry, overflow=False)
        return result

    def _alu_sar(self, a: int, shift: int) -> int:
        """Arithmetic (signed) right shift - sign-extends."""
        a = _to_unsigned32(a)
        shift = shift & 31
        signed_a = _to_signed32(a)
        if shift == 0:
            result = a
            self.flags.update(result, carry=False, overflow=False)
            return result
        carry = bool((a >> (shift - 1)) & 1)
        result = _to_unsigned32(signed_a >> shift)  # Python >> preserves sign
        self.flags.update(result, carry=carry, overflow=False)
        return result

    # ------------------------------------------------------------------
    # Instruction decoder / executor
    # ------------------------------------------------------------------

    def _fetch32(self) -> int:
        val = self.mem_read32(self.pc)
        self.pc = _to_unsigned32(self.pc + 4)
        return val

    def step(self) -> None:
        """Fetch and execute one instruction."""
        if self.halted:
            return

        word0 = self._fetch32()
        opcode = (word0 >> 24) & 0xFF
        rdst   = (word0 >> 20) & 0xF
        ra     = (word0 >> 16) & 0xF
        rb     = (word0 >> 12) & 0xF
        imm12  = word0 & 0xFFF
        # Sign-extend imm12 to 32 bits
        if imm12 & 0x800:
            imm12 = imm12 - 0x1000

        reg = self.registers

        if opcode == OP_NOP:
            pass

        elif opcode == OP_LOAD:
            # LOAD Rdst, [addr32]  — address in next word
            addr = self._fetch32()
            reg[rdst] = self.mem_read32(addr)

        elif opcode == OP_STORE:
            # STORE [addr32], Rsrc — address in next word
            addr = self._fetch32()
            self.mem_write32(addr, reg[ra])

        elif opcode == OP_MOV:
            # rb == 0xF: load next 32-bit word as immediate value
            # rb == anything else: copy reg[ra] to rdst
            if rb == 0xF:
                reg[rdst] = self._fetch32()
            else:
                reg[rdst] = _to_unsigned32(reg[ra])

        elif opcode == OP_ADD:
            # rb == 0xF means use sign-extended imm12 as operand B
            b = imm12 if rb == 0xF else reg[rb]
            reg[rdst] = self._alu_add(reg[ra], b)

        elif opcode == OP_SUB:
            b = imm12 if rb == 0xF else reg[rb]
            reg[rdst] = self._alu_sub(reg[ra], b)

        elif opcode == OP_MUL:
            reg[rdst] = self._alu_mul(reg[ra], reg[rb])

        elif opcode == OP_DIV:
            reg[rdst] = self._alu_div(reg[ra], reg[rb])

        elif opcode == OP_AND:
            b = imm12 if rb == 0xF else reg[rb]
            reg[rdst] = self._alu_and(reg[ra], b)

        elif opcode == OP_OR:
            b = imm12 if rb == 0xF else reg[rb]
            reg[rdst] = self._alu_or(reg[ra], b)

        elif opcode == OP_XOR:
            b = imm12 if rb == 0xF else reg[rb]
            reg[rdst] = self._alu_xor(reg[ra], b)

        elif opcode == OP_NOT:
            reg[rdst] = self._alu_not(reg[ra])

        elif opcode == OP_SHL:
            shift = (imm12 & 31) if rb == 0xF else reg[rb]
            reg[rdst] = self._alu_shl(reg[ra], shift)

        elif opcode == OP_SHR:
            shift = (imm12 & 31) if rb == 0xF else reg[rb]
            reg[rdst] = self._alu_shr(reg[ra], shift)

        elif opcode == OP_SAR:
            shift = (imm12 & 31) if rb == 0xF else reg[rb]
            reg[rdst] = self._alu_sar(reg[ra], shift)

        elif opcode == OP_CMP:
            b = imm12 if rb == 0xF else reg[rb]
            self._alu_sub(reg[ra], b)  # flags only, discard result

        elif opcode == OP_JMP:
            self.pc = self._fetch32()

        elif opcode == OP_JEQ:
            target = self._fetch32()
            if self.flags.zero:
                self.pc = target

        elif opcode == OP_JNE:
            target = self._fetch32()
            if not self.flags.zero:
                self.pc = target

        elif opcode == OP_JLT:
            target = self._fetch32()
            if self.flags.sign != self.flags.overflow:
                self.pc = target

        elif opcode == OP_JGT:
            target = self._fetch32()
            if not self.flags.zero and (self.flags.sign == self.flags.overflow):
                self.pc = target

        elif opcode == OP_JLE:
            target = self._fetch32()
            if self.flags.zero or (self.flags.sign != self.flags.overflow):
                self.pc = target

        elif opcode == OP_JGE:
            target = self._fetch32()
            if self.flags.sign == self.flags.overflow:
                self.pc = target

        elif opcode == OP_CALL:
            target = self._fetch32()
            self._push(self.pc)
            self.pc = target

        elif opcode == OP_RET:
            self.pc = self._pop()

        elif opcode == OP_PUSH:
            self._push(reg[ra])

        elif opcode == OP_POP:
            reg[rdst] = self._pop()

        elif opcode == OP_HALT:
            self.halted = True

        else:
            raise ValueError(f"Unknown opcode: 0x{opcode:02X} at PC=0x{self.pc - 4:08X}")

    def run(self, max_steps: int = 1_000_000) -> int:
        """Run until HALT or max_steps, returning the step count."""
        steps = 0
        while not self.halted and steps < max_steps:
            self.step()
            steps += 1
        return steps

    def load_program(self, program: list[int], start: int = 0) -> None:
        """Load a list of 32-bit words into memory at *start*."""
        for i, word in enumerate(program):
            self.mem_write32(start + i * 4, _to_unsigned32(word))
        self.pc = start

    def dump_registers(self) -> str:
        lines = []
        for i, v in enumerate(self.registers):
            name = f"R{i:02d}" if i < SP else "SP "
            lines.append(f"{name} = 0x{v:08X} ({_to_signed32(v):12d})")
        lines.append(f"PC  = 0x{self.pc:08X}")
        lines.append(str(self.flags))
        return "\n".join(lines)
