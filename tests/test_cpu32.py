"""Tests for the 32Hybrid CPU emulator and assembler."""

import pytest

from cpu32 import (
    CPU, Flags, WORD_MASK, SIGN_BIT,
    _to_signed32, _to_unsigned32,
    OP_NOP, OP_HALT, OP_MOV, OP_ADD, OP_SUB, OP_MUL, OP_DIV,
    OP_AND, OP_OR, OP_XOR, OP_NOT, OP_SHL, OP_SHR, OP_SAR,
    OP_CMP, OP_JMP, OP_JEQ, OP_JNE, OP_JLT, OP_JGT, OP_JLE, OP_JGE,
    OP_CALL, OP_RET, OP_PUSH, OP_POP, OP_LOAD, OP_STORE,
    SP, MEM_SIZE,
)
from assembler import assemble, AssemblerError


# ============================================================
# Helpers
# ============================================================

def make_cpu() -> CPU:
    return CPU(memory_size=MEM_SIZE)


def run_program(program: list[int]) -> CPU:
    cpu = make_cpu()
    cpu.load_program(program)
    cpu.run()
    return cpu


def asm_run(source: str) -> CPU:
    """Assemble source and run it, returning the CPU."""
    program = assemble(source)
    return run_program(program)


def word(opcode: int, rdst: int = 0, ra: int = 0, rb: int = 0, imm12: int = 0) -> int:
    imm12 &= 0xFFF
    return ((opcode & 0xFF) << 24) | ((rdst & 0xF) << 20) | ((ra & 0xF) << 16) | ((rb & 0xF) << 12) | imm12


# ============================================================
# Utility function tests
# ============================================================

class TestUtils:
    def test_to_signed32_positive(self):
        assert _to_signed32(0) == 0
        assert _to_signed32(1) == 1
        assert _to_signed32(0x7FFFFFFF) == 2147483647

    def test_to_signed32_negative(self):
        assert _to_signed32(0xFFFFFFFF) == -1
        assert _to_signed32(0x80000000) == -2147483648
        assert _to_signed32(0xFFFFFFFE) == -2

    def test_to_signed32_masks(self):
        # Should mask to 32 bits before interpreting
        assert _to_signed32(0x1FFFFFFFF) == -1

    def test_to_unsigned32(self):
        assert _to_unsigned32(0) == 0
        assert _to_unsigned32(-1) == 0xFFFFFFFF
        assert _to_unsigned32(-2) == 0xFFFFFFFE
        assert _to_unsigned32(0x100000000) == 0


# ============================================================
# Flags tests
# ============================================================

class TestFlags:
    def test_zero_flag(self):
        f = Flags()
        f.update(0)
        assert f.zero is True
        f.update(1)
        assert f.zero is False

    def test_sign_flag(self):
        f = Flags()
        f.update(0x80000000)
        assert f.sign is True
        f.update(0x7FFFFFFF)
        assert f.sign is False

    def test_carry_and_overflow(self):
        f = Flags()
        f.update(0, carry=True, overflow=True)
        assert f.carry is True
        assert f.overflow is True


# ============================================================
# Memory tests
# ============================================================

class TestMemory:
    def test_read_write_roundtrip(self):
        cpu = make_cpu()
        for addr in [0, 4, 100, MEM_SIZE - 4]:
            cpu.mem_write32(addr, 0xDEADBEEF)
            assert cpu.mem_read32(addr) == 0xDEADBEEF

    def test_little_endian(self):
        cpu = make_cpu()
        cpu.mem_write32(0, 0x01020304)
        assert cpu.memory[0] == 0x04
        assert cpu.memory[1] == 0x03
        assert cpu.memory[2] == 0x02
        assert cpu.memory[3] == 0x01

    def test_byte_access(self):
        cpu = make_cpu()
        cpu.mem_write8(0, 0xAB)
        assert cpu.mem_read8(0) == 0xAB

    def test_out_of_bounds_raises(self):
        cpu = make_cpu()
        with pytest.raises(MemoryError):
            cpu.mem_read32(MEM_SIZE)
        with pytest.raises(MemoryError):
            cpu.mem_write32(MEM_SIZE, 0)

    def test_wrap_value_on_write(self):
        cpu = make_cpu()
        cpu.mem_write32(0, 0x1FFFFFFFF)  # Should be masked to 0xFFFFFFFF
        assert cpu.mem_read32(0) == 0xFFFFFFFF


# ============================================================
# ALU tests (via CPU internal methods)
# ============================================================

class TestALU:
    def setup_method(self):
        self.cpu = make_cpu()

    def test_add_basic(self):
        assert self.cpu._alu_add(1, 2) == 3
        assert not self.cpu.flags.carry
        assert not self.cpu.flags.overflow

    def test_add_unsigned_overflow_sets_carry(self):
        result = self.cpu._alu_add(0xFFFFFFFF, 1)
        assert result == 0
        assert self.cpu.flags.carry
        assert self.cpu.flags.zero

    def test_add_signed_overflow(self):
        # 0x7FFFFFFF + 1 should set overflow
        result = self.cpu._alu_add(0x7FFFFFFF, 1)
        assert result == 0x80000000
        assert self.cpu.flags.overflow
        assert self.cpu.flags.sign

    def test_add_no_overflow_negative_plus_negative(self):
        # -1 + -1 = -2, no signed overflow
        result = self.cpu._alu_add(0xFFFFFFFF, 0xFFFFFFFF)
        assert result == 0xFFFFFFFE
        assert not self.cpu.flags.overflow

    def test_sub_basic(self):
        assert self.cpu._alu_sub(5, 3) == 2
        assert not self.cpu.flags.carry
        assert not self.cpu.flags.overflow

    def test_sub_borrow(self):
        result = self.cpu._alu_sub(0, 1)
        assert result == 0xFFFFFFFF
        assert self.cpu.flags.carry  # borrow

    def test_sub_signed_overflow(self):
        # 0x80000000 (-2147483648) - 1 = 0x7FFFFFFF; signed overflow
        result = self.cpu._alu_sub(0x80000000, 1)
        assert result == 0x7FFFFFFF
        assert self.cpu.flags.overflow

    def test_mul_basic(self):
        assert self.cpu._alu_mul(6, 7) == 42
        assert self.cpu._alu_mul(0xFFFF, 0xFFFF) == (0xFFFF * 0xFFFF) & WORD_MASK

    def test_mul_truncates_to_32_bits(self):
        result = self.cpu._alu_mul(0xFFFFFFFF, 2)
        assert result == 0xFFFFFFFE

    def test_div_basic(self):
        assert self.cpu._alu_div(10, 3) == 3
        assert self.cpu._alu_div(100, 10) == 10

    def test_div_by_zero_raises(self):
        with pytest.raises(ZeroDivisionError):
            self.cpu._alu_div(1, 0)

    def test_and(self):
        assert self.cpu._alu_and(0xFF00, 0x0FF0) == 0x0F00

    def test_or(self):
        assert self.cpu._alu_or(0xFF00, 0x00FF) == 0xFFFF

    def test_xor(self):
        assert self.cpu._alu_xor(0xFFFF, 0x0F0F) == 0xF0F0

    def test_not(self):
        assert self.cpu._alu_not(0) == 0xFFFFFFFF
        assert self.cpu._alu_not(0xFFFFFFFF) == 0

    def test_shl_basic(self):
        assert self.cpu._alu_shl(1, 4) == 16
        assert self.cpu._alu_shl(0x80000000, 1) == 0  # Shifted out

    def test_shl_carry(self):
        self.cpu._alu_shl(0x80000000, 1)
        assert self.cpu.flags.carry  # High bit shifted out into carry

    def test_shl_clamps_shift(self):
        # shift & 31 should clamp 32+ shifts
        assert self.cpu._alu_shl(1, 32) == self.cpu._alu_shl(1, 0)

    def test_shr_basic(self):
        assert self.cpu._alu_shr(16, 4) == 1

    def test_shr_does_not_sign_extend(self):
        # Logical shift: high bit becomes 0
        result = self.cpu._alu_shr(0x80000000, 1)
        assert result == 0x40000000

    def test_sar_sign_extends(self):
        # Arithmetic shift: high bit is replicated
        result = self.cpu._alu_sar(0x80000000, 1)
        assert result == 0xC0000000

    def test_sar_positive(self):
        assert self.cpu._alu_sar(16, 4) == 1


# ============================================================
# Instruction execution tests
# ============================================================

class TestInstructions:
    def test_nop(self):
        prog = [
            word(OP_NOP),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.halted

    def test_halt(self):
        prog = [word(OP_HALT)]
        cpu = run_program(prog)
        assert cpu.halted

    def test_mov_reg_to_reg(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF),  # MOV R0, #42 (imm32)
            42,
            word(OP_MOV, rdst=1, ra=0, rb=0),     # MOV R1, R0
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[0] == 42
        assert cpu.registers[1] == 42

    def test_mov_imm32(self):
        prog = [
            word(OP_MOV, rdst=3, ra=0, rb=0xF),  # MOV R3, #0xDEADBEEF
            0xDEADBEEF,
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[3] == 0xDEADBEEF

    def test_add_registers(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 10,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 20,
            word(OP_ADD, rdst=2, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[2] == 30

    def test_sub_registers(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 100,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 37,
            word(OP_SUB, rdst=2, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[2] == 63

    def test_mul(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 6,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 7,
            word(OP_MUL, rdst=2, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[2] == 42

    def test_div(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 42,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 6,
            word(OP_DIV, rdst=2, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[2] == 7

    def test_and(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 0xFF00FF00,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 0x0F0F0F0F,
            word(OP_AND, rdst=2, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[2] == 0x0F000F00

    def test_or(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 0xF0F0F0F0,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 0x0F0F0F0F,
            word(OP_OR, rdst=2, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[2] == 0xFFFFFFFF

    def test_xor_self_zeroes(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 0x12345678,
            word(OP_XOR, rdst=0, ra=0, rb=0),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[0] == 0

    def test_not(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 0,
            word(OP_NOT, rdst=1, ra=0),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[1] == 0xFFFFFFFF

    def test_shl(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 1,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 8,
            word(OP_SHL, rdst=2, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[2] == 256

    def test_shr(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 0x80000000,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 1,
            word(OP_SHR, rdst=2, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[2] == 0x40000000  # No sign extension

    def test_sar(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 0x80000000,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 1,
            word(OP_SAR, rdst=2, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[2] == 0xC0000000  # Sign extended

    def test_cmp_sets_zero_flag(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 5,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 5,
            word(OP_CMP, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.flags.zero

    def test_load_store(self):
        data_addr = 0x8000
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 0xCAFEBABE,
            word(OP_STORE, ra=0), data_addr,
            word(OP_LOAD, rdst=1), data_addr,
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[1] == 0xCAFEBABE

    def test_jmp(self):
        # Jump skips over a NOP to HALT
        prog_base = 0
        # Address of HALT: prog_base + 3*4 = 12 (index 3)
        halt_addr = prog_base + 3 * 4
        prog = [
            word(OP_JMP),           # offset 0
            halt_addr,              # offset 4 (jump target)
            word(OP_MOV, rdst=0, ra=0, rb=0xF),  # offset 8 (SKIPPED)
            word(OP_HALT),          # offset 12
        ]
        cpu = run_program(prog)
        assert cpu.registers[0] == 0  # MOV should have been skipped

    def test_jeq_taken(self):
        # Set up: R0==R1, then JEQ should jump over an instruction
        prog_base = 0
        halt_addr = prog_base + 7 * 4  # instruction 7
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 7,   # 0,1
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 7,   # 2,3
            word(OP_CMP, ra=0, rb=1),                 # 4
            word(OP_JEQ), halt_addr,                  # 5,6
            word(OP_MOV, rdst=2, ra=0, rb=0xF), 99,  # 7,8 (skipped)
            word(OP_HALT),                             # 9 -> wrong, recalculate
        ]
        # halt at index 9 = address 36, skip at indices 7-8
        halt_addr2 = 9 * 4
        prog[6] = halt_addr2
        cpu = run_program(prog)
        assert cpu.registers[2] == 0  # skipped

    def test_jne_not_taken(self):
        prog_base = 0
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 5,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 5,
            word(OP_CMP, ra=0, rb=1),
            word(OP_JNE), 100 * 4,  # should NOT jump (equal)
            word(OP_MOV, rdst=2, ra=0, rb=0xF), 42,
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[2] == 42

    def test_push_pop(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 0x1234ABCD,
            word(OP_PUSH, ra=0),
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 0,   # Clear R0
            word(OP_POP, rdst=0),
            word(OP_HALT),
        ]
        cpu = run_program(prog)
        assert cpu.registers[0] == 0x1234ABCD

    def test_call_ret(self):
        # Program layout (word indices / addresses):
        # 0: JMP to main (skip subroutine)
        # 1: address of main = index 4 = 16
        # 2: subroutine: MOV R0, #0xBEEF (two words)
        # 3:   imm32
        # 4: subroutine: RET
        # -- main --
        # 5: NOP (this is index 5 = address 20 = target of JMP)
        # Wait, let me lay out correctly...

        sub_addr  = 2 * 4   # subroutine starts at index 2 (address 8)
        main_addr = 5 * 4   # main starts at index 5 (address 20)
        prog = [
            word(OP_JMP),              # 0 -> jump to main
            main_addr,                 # 1
            # subroutine
            word(OP_MOV, rdst=0, ra=0, rb=0xF),  # 2
            0xBEEF,                    # 3
            word(OP_RET),              # 4
            # main
            word(OP_CALL),             # 5
            sub_addr,                  # 6
            word(OP_HALT),             # 7
        ]
        cpu = run_program(prog)
        assert cpu.registers[0] == 0xBEEF

    def test_unknown_opcode_raises(self):
        prog = [0xFF000000]  # opcode 0xFF is undefined
        cpu = make_cpu()
        cpu.load_program(prog)
        with pytest.raises(ValueError, match="Unknown opcode"):
            cpu.step()

    def test_div_by_zero_raises(self):
        prog = [
            word(OP_MOV, rdst=0, ra=0, rb=0xF), 10,
            word(OP_MOV, rdst=1, ra=0, rb=0xF), 0,
            word(OP_DIV, rdst=2, ra=0, rb=1),
            word(OP_HALT),
        ]
        cpu = make_cpu()
        cpu.load_program(prog)
        with pytest.raises(ZeroDivisionError):
            cpu.run()

    def test_max_steps_stops_runaway(self):
        # Infinite loop
        prog = [
            word(OP_JMP), 0,  # Always jump back to 0
        ]
        cpu = make_cpu()
        cpu.load_program(prog)
        steps = cpu.run(max_steps=100)
        assert steps == 100
        assert not cpu.halted


# ============================================================
# Assembler tests
# ============================================================

class TestAssembler:
    def test_nop_halt(self):
        prog = assemble("NOP\nHALT")
        assert len(prog) == 2

    def test_mov_imm_small(self):
        prog = assemble("MOV R0, #42\nHALT")
        cpu = run_program(prog)
        assert cpu.registers[0] == 42

    def test_mov_imm_large(self):
        prog = assemble("MOV R0, #0xDEADBEEF\nHALT")
        cpu = run_program(prog)
        assert cpu.registers[0] == 0xDEADBEEF

    def test_mov_negative_imm(self):
        prog = assemble("MOV R0, #-1\nHALT")
        cpu = run_program(prog)
        assert cpu.registers[0] == 0xFFFFFFFF

    def test_add_reg_reg(self):
        source = """
        MOV R1, #100
        MOV R2, #200
        ADD R0, R1, R2
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 300

    def test_add_reg_imm(self):
        source = """
        MOV R1, #50
        ADD R0, R1, #25
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 75

    def test_sub(self):
        source = """
        MOV R1, #200
        SUB R0, R1, #150
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 50

    def test_mul_asm(self):
        source = """
        MOV R1, #12
        MOV R2, #11
        MUL R0, R1, R2
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 132

    def test_div_asm(self):
        source = """
        MOV R1, #100
        MOV R2, #4
        DIV R0, R1, R2
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 25

    def test_and_asm(self):
        source = """
        MOV R1, #0xFF
        AND R0, R1, #0x0F
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 0x0F

    def test_or_asm(self):
        source = """
        MOV R1, #0xF0
        OR R0, R1, #0x0F
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 0xFF

    def test_xor_asm(self):
        source = """
        MOV R1, #0xFF
        XOR R0, R1, #0xF0
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 0x0F

    def test_not_asm(self):
        source = """
        MOV R1, #0
        NOT R0, R1
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 0xFFFFFFFF

    def test_shl_asm(self):
        source = """
        MOV R1, #1
        SHL R0, R1, #8
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 256

    def test_shr_asm(self):
        source = """
        MOV R1, #0x100
        SHR R0, R1, #4
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 16

    def test_sar_asm(self):
        source = """
        MOV R1, #0x80000000
        SAR R0, R1, #4
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 0xF8000000

    def test_cmp_jmp(self):
        source = """
        MOV R1, #10
        MOV R2, #10
        CMP R1, R2
        JEQ done
        MOV R0, #99   ; should not execute
done:
        MOV R0, #1
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 1

    def test_loop_countdown(self):
        """Count down from 5 to 0 using a loop."""
        source = """
        MOV R0, #5
loop:
        SUB R0, R0, #1
        CMP R0, #0
        JNE loop
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 0

    def test_load_store_asm(self):
        source = """
        MOV R0, #0xABCD1234
        STORE [0x8000], R0
        LOAD R1, [0x8000]
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[1] == 0xABCD1234

    def test_push_pop_asm(self):
        source = """
        MOV R0, #0xFACE
        PUSH R0
        MOV R0, #0
        POP R0
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 0xFACE

    def test_call_ret_asm(self):
        source = """
        CALL double
        HALT
double:
        MOV R0, #42
        RET
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 42

    def test_forward_label(self):
        source = """
        JMP end
        MOV R0, #99
end:
        MOV R0, #1
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 1

    def test_comment_stripped(self):
        prog = assemble("; full line comment\nNOP ; inline comment\nHALT")
        assert len(prog) == 2

    def test_word_directive(self):
        prog = assemble(".word 0xDEADBEEF")
        assert len(prog) == 1
        assert prog[0] == 0xDEADBEEF

    def test_unknown_mnemonic_raises(self):
        with pytest.raises(AssemblerError, match="Unknown mnemonic"):
            assemble("FOOBAR R0")

    def test_undefined_label_raises(self):
        with pytest.raises(AssemblerError, match="Undefined label"):
            assemble("JMP nowhere")

    def test_duplicate_label_raises(self):
        with pytest.raises(AssemblerError, match="duplicate label"):
            assemble("x:\nx:\nHALT")

    def test_sp_register_name(self):
        source = """
        MOV SP, #0xFFFC
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[SP] == 0xFFFC

    def test_jlt_signed(self):
        """JLT should jump when Ra < Rb (signed)."""
        source = """
        MOV R1, #0xFFFFFFFF   ; -1 in two's complement
        MOV R2, #0
        CMP R1, R2
        JLT negative
        MOV R0, #0
        HALT
negative:
        MOV R0, #1
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 1

    def test_jge_signed(self):
        source = """
        MOV R1, #5
        MOV R2, #3
        CMP R1, R2
        JGE greater
        MOV R0, #0
        HALT
greater:
        MOV R0, #1
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 1


# ============================================================
# Integration tests  — slightly more complex programs
# ============================================================

class TestIntegration:
    def test_fibonacci(self):
        """Compute Fibonacci(10) = 55."""
        source = """
        MOV R0, #0       ; fib(n-2)
        MOV R1, #1       ; fib(n-1)
        MOV R2, #10      ; counter
loop:
        ADD R3, R0, R1   ; R3 = fib(n)
        MOV R0, R1       ; fib(n-2) <- fib(n-1)
        MOV R1, R3       ; fib(n-1) <- fib(n)
        SUB R2, R2, #1
        CMP R2, #0
        JNE loop
        ; After 10 iterations, R0 holds fib(10) = 55
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 55

    def test_factorial(self):
        """Compute 6! = 720."""
        source = """
        MOV R0, #1       ; result
        MOV R1, #6       ; counter
loop:
        MUL R0, R0, R1
        SUB R1, R1, #1
        CMP R1, #0
        JNE loop
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 720

    def test_sum_1_to_100(self):
        """Sum integers 1..100 = 5050."""
        source = """
        MOV R0, #0       ; accumulator
        MOV R1, #1       ; counter
loop:
        ADD R0, R0, R1
        ADD R1, R1, #1
        CMP R1, #101
        JLT loop
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 5050

    def test_32bit_arithmetic_no_overflow(self):
        """Verify 32-bit arithmetic wraps correctly and doesn't corrupt state."""
        source = """
        MOV R0, #0xFFFFFFFF
        ADD R0, R0, #1      ; Should wrap to 0 with carry
        HALT
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 0
        assert cpu.flags.carry

    def test_nested_calls(self):
        """Test call/ret with nested subroutines."""
        source = """
        CALL outer
        HALT

outer:
        CALL inner
        ADD R0, R0, #10
        RET

inner:
        MOV R0, #5
        RET
        """
        cpu = asm_run(source)
        assert cpu.registers[0] == 15

    def test_dump_registers(self):
        prog = assemble("MOV R0, #0xABCD\nHALT")
        cpu = run_program(prog)
        dump = cpu.dump_registers()
        assert "R00" in dump
        assert "0x0000ABCD" in dump
