"""
32Hybrid Assembler
Converts human-readable assembly into a list of 32-bit instruction words
suitable for loading into the 32Hybrid CPU emulator.

Instruction formats
-------------------
Encoding uses 32-bit fixed-width words.  Bits [31:24] hold the opcode;
the remaining bits carry register indices and immediate data.

  [31:24]  opcode   (8 bits)
  [23:20]  Rdst     (4 bits)
  [19:16]  Ra       (4 bits)
  [15:12]  Rb / 0   (4 bits)  — 0xF sentinel means "use imm32 follows"
  [11:0]   imm12    (12 bits, sign-extended)

Instructions that take a 32-bit immediate or address emit a second word.

Syntax
------
  ; line comment
  label:
  NOP
  HALT
  MOV  R1, R2         ; register copy
  MOV  R1, #42        ; load small immediate (sign-extended 12-bit)
  MOV  R1, #0xDEAD    ; load 32-bit immediate (two-word encoding)
  ADD  R0, R1, R2
  ADD  R0, R1, #5
  SUB  R0, R1, R2
  SUB  R0, R1, #5
  MUL  R0, R1, R2
  DIV  R0, R1, R2
  AND  R0, R1, R2
  AND  R0, R1, #0xFF
  OR   R0, R1, R2
  XOR  R0, R1, R2
  NOT  R0, R1
  SHL  R0, R1, R2
  SHL  R0, R1, #3
  SHR  R0, R1, R2
  SAR  R0, R1, R2
  CMP  R1, R2
  CMP  R1, #0
  LOAD  R0, [0x1000]
  STORE [0x1000], R0
  JMP  label / #addr
  JEQ  label / #addr
  JNE  label / #addr
  JLT  label / #addr
  JGT  label / #addr
  JLE  label / #addr
  JGE  label / #addr
  CALL label / #addr
  RET
  PUSH R0
  POP  R0
  .word 0xDEADBEEF    ; raw 32-bit literal
"""

from __future__ import annotations

import re
from typing import Union

from cpu32 import (
    WORD_MASK,
    OP_NOP, OP_LOAD, OP_STORE, OP_MOV,
    OP_ADD, OP_SUB, OP_MUL, OP_DIV,
    OP_AND, OP_OR, OP_XOR, OP_NOT,
    OP_SHL, OP_SHR, OP_SAR,
    OP_CMP,
    OP_JMP, OP_JEQ, OP_JNE, OP_JLT, OP_JGT, OP_JLE, OP_JGE,
    OP_CALL, OP_RET,
    OP_PUSH, OP_POP,
    OP_HALT,
    SP,
)

# Symbolic register names
_REG_NAMES: dict[str, int] = {f"R{i}": i for i in range(16)}
_REG_NAMES["SP"] = SP

# Branch / jump opcodes
_BRANCH_OPS: dict[str, int] = {
    "JMP": OP_JMP,
    "JEQ": OP_JEQ,
    "JNE": OP_JNE,
    "JLT": OP_JLT,
    "JGT": OP_JGT,
    "JLE": OP_JLE,
    "JGE": OP_JGE,
    "CALL": OP_CALL,
}

# ALU ops that take (Rdst, Ra, Rb/imm)
_ALU3_OPS: dict[str, int] = {
    "ADD": OP_ADD,
    "SUB": OP_SUB,
    "AND": OP_AND,
    "OR":  OP_OR,
    "XOR": OP_XOR,
    "SHL": OP_SHL,
    "SHR": OP_SHR,
    "SAR": OP_SAR,
}


def _parse_int(token: str) -> int:
    """Parse a decimal, hex, or binary integer token."""
    token = token.strip()
    if token.startswith("0x") or token.startswith("0X"):
        return int(token, 16)
    if token.startswith("0b") or token.startswith("0B"):
        return int(token, 2)
    return int(token, 10)


def _parse_reg(token: str) -> int:
    """Parse a register name like R0..R15 or SP."""
    token = token.strip().upper()
    if token not in _REG_NAMES:
        raise ValueError(f"Unknown register: {token!r}")
    return _REG_NAMES[token]


def _parse_imm(token: str) -> int:
    """Parse an immediate value; the leading # is optional."""
    token = token.strip()
    if token.startswith("#"):
        token = token[1:]
    return _parse_int(token)


def _encode_word(opcode: int, rdst: int = 0, ra: int = 0, rb: int = 0, imm12: int = 0) -> int:
    """Pack the standard 32-bit instruction word."""
    imm12 &= 0xFFF
    return (
        ((opcode & 0xFF) << 24)
        | ((rdst & 0xF) << 20)
        | ((ra & 0xF) << 16)
        | ((rb & 0xF) << 12)
        | (imm12 & 0xFFF)
    )


class AssemblerError(Exception):
    pass


# Each "pending" entry is a placeholder index + label name for forward refs.
_Pending = tuple[int, str]


def assemble(source: str, origin: int = 0) -> list[int]:
    """
    Assemble *source* into a list of 32-bit instruction words.

    *origin* is the byte address where the first word will be loaded;
    it is used to resolve label addresses correctly.
    """
    # ------------------------------------------------------------------ pass 1
    # Tokenise and emit placeholder words, collecting label positions.
    labels: dict[str, int] = {}
    # Each element is either an int (resolved word) or a pending tuple.
    raw: list[Union[int, _Pending]] = []

    def current_byte_addr() -> int:
        return origin + len(raw) * 4

    lines = source.splitlines()
    for lineno, line in enumerate(lines, 1):
        # Strip comments
        line = line.split(";")[0].strip()
        if not line:
            continue

        # Label definition
        if line.endswith(":"):
            label = line[:-1].strip()
            if not label.isidentifier():
                raise AssemblerError(f"Line {lineno}: invalid label {label!r}")
            if label in labels:
                raise AssemblerError(f"Line {lineno}: duplicate label {label!r}")
            labels[label] = current_byte_addr()
            continue

        parts = re.split(r"[\s,]+", line, maxsplit=1)
        mnemonic = parts[0].upper()
        operands_str = parts[1].strip() if len(parts) > 1 else ""

        # Split operands on comma (but not inside brackets)
        operands = [o.strip() for o in re.split(r",\s*(?![^\[]*\])", operands_str) if o.strip()]

        try:
            _assemble_line(mnemonic, operands, raw, labels, current_byte_addr, lineno)
        except (AssemblerError, ValueError) as exc:
            raise AssemblerError(f"Line {lineno}: {exc}") from exc

    # ------------------------------------------------------------------ pass 2
    # Resolve forward references.
    resolved: list[int] = []
    for i, item in enumerate(raw):
        if isinstance(item, int):
            resolved.append(item & WORD_MASK)
        else:
            label = item[1]
            if label not in labels:
                raise AssemblerError(f"Undefined label: {label!r}")
            resolved.append(labels[label] & WORD_MASK)
    return resolved


def _assemble_line(
    mnemonic: str,
    operands: list[str],
    raw: list,
    labels: dict[str, int],
    current_byte_addr,
    lineno: int,
) -> None:
    """Encode a single instruction and append words to *raw*."""

    def _resolve_target(token: str) -> None:
        """Append a target address word (may be a label for forward ref)."""
        if token.startswith("#"):
            raw.append(_parse_imm(token))
        elif token in labels:
            raw.append(labels[token])
        else:
            # Forward reference placeholder
            raw.append((lineno, token))

    if mnemonic == "NOP":
        raw.append(_encode_word(OP_NOP))

    elif mnemonic == "HALT":
        raw.append(_encode_word(OP_HALT))

    elif mnemonic == "RET":
        raw.append(_encode_word(OP_RET))

    elif mnemonic == ".WORD":
        if len(operands) != 1:
            raise AssemblerError(".word requires exactly one operand")
        raw.append(_parse_imm(operands[0]))

    elif mnemonic == "MOV":
        if len(operands) != 2:
            raise AssemblerError("MOV requires 2 operands")
        rdst = _parse_reg(operands[0])
        src = operands[1]
        if src.startswith("#") or src.lstrip("-").isdigit():
            imm = _parse_imm(src)
            # Always use the two-word 32-bit immediate encoding (rb=0xF sentinel)
            raw.append(_encode_word(OP_MOV, rdst=rdst, ra=0, rb=0xF))
            raw.append(imm & WORD_MASK)
        else:
            ra = _parse_reg(src)
            raw.append(_encode_word(OP_MOV, rdst=rdst, ra=ra, rb=0))

    elif mnemonic in _ALU3_OPS:
        opcode = _ALU3_OPS[mnemonic]
        if len(operands) != 3:
            raise AssemblerError(f"{mnemonic} requires 3 operands")
        rdst = _parse_reg(operands[0])
        ra   = _parse_reg(operands[1])
        b_tok = operands[2]
        if b_tok.startswith("#"):
            imm = _parse_imm(b_tok)
            # rb=0xF is the sentinel for "use sign-extended imm12"
            raw.append(_encode_word(opcode, rdst=rdst, ra=ra, rb=0xF, imm12=imm))
        else:
            rb = _parse_reg(b_tok)
            raw.append(_encode_word(opcode, rdst=rdst, ra=ra, rb=rb))

    elif mnemonic == "MUL":
        if len(operands) != 3:
            raise AssemblerError("MUL requires 3 operands")
        rdst = _parse_reg(operands[0])
        ra   = _parse_reg(operands[1])
        rb   = _parse_reg(operands[2])
        raw.append(_encode_word(OP_MUL, rdst=rdst, ra=ra, rb=rb))

    elif mnemonic == "DIV":
        if len(operands) != 3:
            raise AssemblerError("DIV requires 3 operands")
        rdst = _parse_reg(operands[0])
        ra   = _parse_reg(operands[1])
        rb   = _parse_reg(operands[2])
        raw.append(_encode_word(OP_DIV, rdst=rdst, ra=ra, rb=rb))

    elif mnemonic == "NOT":
        if len(operands) != 2:
            raise AssemblerError("NOT requires 2 operands")
        rdst = _parse_reg(operands[0])
        ra   = _parse_reg(operands[1])
        raw.append(_encode_word(OP_NOT, rdst=rdst, ra=ra))

    elif mnemonic == "CMP":
        if len(operands) != 2:
            raise AssemblerError("CMP requires 2 operands")
        ra  = _parse_reg(operands[0])
        b_tok = operands[1]
        if b_tok.startswith("#"):
            imm = _parse_imm(b_tok)
            # rb=0xF is the sentinel for "use sign-extended imm12"
            raw.append(_encode_word(OP_CMP, ra=ra, rb=0xF, imm12=imm))
        else:
            rb = _parse_reg(b_tok)
            raw.append(_encode_word(OP_CMP, ra=ra, rb=rb))

    elif mnemonic == "LOAD":
        if len(operands) != 2:
            raise AssemblerError("LOAD requires 2 operands")
        rdst = _parse_reg(operands[0])
        # Strip brackets from address operand
        addr_tok = operands[1].strip().lstrip("[").rstrip("]")
        raw.append(_encode_word(OP_LOAD, rdst=rdst))
        raw.append(_parse_imm(addr_tok))

    elif mnemonic == "STORE":
        if len(operands) != 2:
            raise AssemblerError("STORE requires 2 operands")
        # Operands: [addr], Rsrc
        addr_tok = operands[0].strip().lstrip("[").rstrip("]")
        ra = _parse_reg(operands[1])
        raw.append(_encode_word(OP_STORE, ra=ra))
        raw.append(_parse_imm(addr_tok))

    elif mnemonic in _BRANCH_OPS:
        opcode = _BRANCH_OPS[mnemonic]
        if len(operands) != 1:
            raise AssemblerError(f"{mnemonic} requires 1 operand")
        raw.append(_encode_word(opcode))
        _resolve_target(operands[0])

    elif mnemonic == "PUSH":
        if len(operands) != 1:
            raise AssemblerError("PUSH requires 1 operand")
        ra = _parse_reg(operands[0])
        raw.append(_encode_word(OP_PUSH, ra=ra))

    elif mnemonic == "POP":
        if len(operands) != 1:
            raise AssemblerError("POP requires 1 operand")
        rdst = _parse_reg(operands[0])
        raw.append(_encode_word(OP_POP, rdst=rdst))

    else:
        raise AssemblerError(f"Unknown mnemonic: {mnemonic!r}")
