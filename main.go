package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/goldmane/gemu/cpu"
	"github.com/goldmane/gemu/gemu"
)

func HighByte(a uint16) uint8 {
	h := uint8(a >> 8)
	// fmt.Printf("\n\nhi: %02x\n\n", h)
	return h
}

func LowByte(a uint16) uint8 {
	b := uint8(0xFF & a)
	// fmt.Printf("\n\nlo: %02x\n\n", b)
	return b
}

func PageCrossed(a uint16, b uint16) bool {
	pa := (a & 0xFF) >> 8
	pb := (b & 0xFF) >> 8
	return pa != pb
}

type Instruction struct {
	Opcode uint8
	Label  string
	Length int
	// Cycles      uint8 // this is the return value of the Function
	AddressMode  uint8
	Function     func(cpu *cpu.CPU) (uint8, string)
	PrintDetails func(cpu cpu.CPU, ins Instruction) string
}

var instructions = map[uint8]Instruction{
	0x4C: {Opcode: 0x4C, Label: "JMP", Length: 3, AddressMode: cpu.Absolute, Function: func(cpu *cpu.CPU) (uint8, string) {
		ta, s := cpu.Fetch16()
		cpu.TempAddress = ta
		cpu.SetPC(cpu.TempAddress)
		return 3, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.TempAddress)
	}},
	0xA2: {Opcode: 0xA2, Label: "LDX", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		v, s := cpu.Fetch()
		cpu.X.SetRegister(v)
		cpu.Flags.SetZeroByValue(cpu.X.GetValue())
		cpu.Flags.SetNegative(cpu.X.GetValue())
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0x86: {Opcode: 0x86, Label: "STX", Length: 2, AddressMode: cpu.ZeroPageX, Function: func(cpu *cpu.CPU) (uint8, string) {
		a, s := cpu.Fetch()
		cpu.TempAddress = uint16(a)
		cpu.Store(cpu.TempAddress, cpu.X.GetValue())
		return 3, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%02X = %02X", cpu.TempAddress, cpu.X.GetValue())
	}},
	0x20: {Opcode: 0x86, Label: "JSR", Length: 3, AddressMode: cpu.Absolute, Function: func(cpu *cpu.CPU) (uint8, string) {
		// push the current PC + 2 onto the stack
		pc := cpu.GetPC()
		npc := pc + 1
		hi := HighByte(npc)
		cpu.StackPush(hi)
		lo := LowByte(npc)
		cpu.StackPush(lo)
		// get the target address
		ta, s := cpu.Fetch16()
		cpu.TempAddress = ta
		// go to target
		cpu.SetPC(cpu.TempAddress)
		return 6, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.TempAddress)
	}},
	0xEA: {Opcode: 0x86, Label: "NOP", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		// nothing to do here
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x38: {Opcode: 0xA2, Label: "SEC", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		cpu.Flags.SetFlag(gemu.Carry, true)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0xB0: {Opcode: 0xB0, Label: "BCS", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) (uint8, string) {
		cycles := uint8(2)
		offset, s := cpu.Fetch()
		cpu.TempAddress = cpu.GetPC() + uint16(offset)
		if cpu.Flags.GetFlag(gemu.Carry) {
			cycles += 1
			cpu.SetPC(cpu.TempAddress)
		}
		if PageCrossed(cpu.PrevPC, cpu.TempAddress) {
			cycles += 1
		}
		return cycles, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.TempAddress)
	}},
	0x18: {Opcode: 0xA2, Label: "CLC", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		cpu.Flags.SetFlag(gemu.Carry, false)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x90: {Opcode: 0xA2, Label: "BCC", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) (uint8, string) {
		cycles := uint8(2)
		offset, s := cpu.Fetch()
		cpu.TempAddress = cpu.GetPC() + uint16(offset)
		if !cpu.Flags.GetFlag(gemu.Carry) {
			cycles += 1
			cpu.SetPC(cpu.TempAddress)
		}
		if PageCrossed(cpu.PrevPC, cpu.TempAddress) {
			cycles += 1
		}
		return cycles, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.TempAddress)
	}},
	0xA9: {Opcode: 0xA2, Label: "LDA", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		ta, s := cpu.Fetch()
		cpu.TempValue = ta
		cpu.A.SetRegister(cpu.TempValue)
		cpu.Flags.SetZeroByValue(cpu.TempValue)
		cpu.Flags.SetNegative(cpu.TempValue)
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0xF0: {Opcode: 0xA2, Label: "BEQ", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) (uint8, string) {
		cycles := uint8(2)
		offset, s := cpu.Fetch()
		cpu.TempAddress = cpu.GetPC() + uint16(offset)
		if cpu.Flags.GetFlag(gemu.Zero) {
			cycles += 1
			cpu.SetPC(cpu.TempAddress)
		}
		if PageCrossed(cpu.PrevPC, cpu.TempAddress) {
			cycles += 1
		}
		return cycles, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.TempAddress)
	}},
	0xD0: {Opcode: 0xD0, Label: "BNE", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) (uint8, string) {
		cycles := uint8(2)
		offset, s := cpu.Fetch()
		cpu.TempAddress = cpu.GetPC() + uint16(offset)
		z := cpu.Flags.GetFlag(gemu.Zero)
		if !z {
			cycles += 1
			cpu.SetPC(cpu.TempAddress)
		}
		if PageCrossed(cpu.PrevPC, cpu.TempAddress) {
			cycles += 1
		}
		return cycles, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.TempAddress)
	}},
	0x85: {Opcode: 0x85, Label: "STA", Length: 2, AddressMode: cpu.ZeroPageA, Function: func(cpu *cpu.CPU) (uint8, string) {
		a, s := cpu.Fetch()
		cpu.TempAddress = uint16(a)
		cpu.TempValue = cpu.FetchAddress(cpu.TempAddress)
		cpu.Store(cpu.TempAddress, cpu.A.GetValue())
		return 3, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%02X = %02X", cpu.TempAddress, cpu.TempValue)
	}},
	0x24: {Opcode: 0x24, Label: "BIT", Length: 2, AddressMode: cpu.ZeroPageA, Function: func(cpu *cpu.CPU) (uint8, string) {
		a, s := cpu.Fetch()              // get the address
		v := cpu.FetchAddress(uint16(a)) // get the value from that address
		cpu.TempValue = uint8(v)
		cpu.TempAddress = uint16(a)
		r := v & cpu.A.GetValue()
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetOverflow(v)
		cpu.Flags.SetNegative(v)
		return 3, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%02X = %02X", cpu.TempAddress, cpu.TempValue)
	}},
	0x70: {Opcode: 0xA2, Label: "BVS", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) (uint8, string) {
		cycles := uint8(2)
		offset, s := cpu.Fetch()
		cpu.TempAddress = cpu.GetPC() + uint16(offset)
		if cpu.Flags.GetFlag(gemu.Overflow) {
			cycles += 1
			cpu.SetPC(cpu.TempAddress)
		}
		if PageCrossed(cpu.PrevPC, cpu.TempAddress) {
			cycles += 1
		}
		return cycles, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.TempAddress)
	}},
	0x50: {Opcode: 0xA2, Label: "BVC", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) (uint8, string) {
		cycles := uint8(2)
		offset, s := cpu.Fetch()
		cpu.TempAddress = cpu.GetPC() + uint16(offset)
		if !cpu.Flags.GetFlag(gemu.Overflow) {
			cycles += 1
			cpu.SetPC(cpu.TempAddress)
		}
		if PageCrossed(cpu.PrevPC, cpu.TempAddress) {
			cycles += 1
		}
		return cycles, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.TempAddress)
	}},
	0x10: {Opcode: 0xA2, Label: "BPL", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) (uint8, string) {
		cycles := uint8(2)
		offset, s := cpu.Fetch()
		cpu.TempAddress = cpu.GetPC() + uint16(offset)
		if !cpu.Flags.GetFlag(gemu.Negative) {
			cycles += 1
			cpu.SetPC(cpu.TempAddress)
		}
		if PageCrossed(cpu.PrevPC, cpu.TempAddress) {
			cycles += 1
		}
		return cycles, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.TempAddress)
	}},
	0x60: {Opcode: 0x60, Label: "RTS", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		lo := cpu.StackPop()
		hi := cpu.StackPop()
		// fmt.Printf("\n\n%02x %02x\n\n", lo, hi)
		cpu.SetPC(ToAddress(hi, lo) + 1)
		return 6, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x78: {Opcode: 0x60, Label: "SEI", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		cpu.Flags.SetFlag(gemu.InterruptDisable, true)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0xF8: {Opcode: 0x60, Label: "SED", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		cpu.Flags.SetFlag(gemu.Decimal, true)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x08: {Opcode: 0x08, Label: "PHP", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		v := cpu.Flags.Value()
		nv := v | 0x30
		cpu.StackPush(nv)
		return 3, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x68: {Opcode: 0x68, Label: "PLA", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		v := cpu.StackPop()
		// cpu.A.SetRegister(v + 0x10)
		cpu.A.SetRegister(v)
		cpu.Flags.SetNegative(v)
		cpu.Flags.SetZeroByValue(v)
		return 4, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x29: {Opcode: 0x26, Label: "AND", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		v, s := cpu.Fetch()
		a := cpu.A.GetValue()
		r := v & a
		cpu.A.SetRegister(r)
		cpu.Flags.SetNegative(r)
		cpu.Flags.SetZeroByValue(r)
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0xC9: {Opcode: 0xC9, Label: "CMP", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		a := cpu.A.GetValue()
		v, s := cpu.Fetch()
		r := a - v
		cpu.Flags.SetFlag(gemu.Carry, a >= v)
		// cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetFlag(gemu.Zero, a == v)
		// cpu.Flags.SetZero(r)
		cpu.Flags.SetNegative(r)
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0xD8: {Opcode: 0xD8, Label: "CLD", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		cpu.Flags.SetFlag(gemu.Decimal, false)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x48: {Opcode: 0x48, Label: "PHA", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		cpu.StackPush(cpu.A.GetValue())
		return 3, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x28: {Opcode: 0x28, Label: "PLP", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		v := cpu.StackPop()
		cpu.Flags.SetCarry(v)
		cpu.Flags.SetZero(v)
		cpu.Flags.SetInterruptDisable(v)
		cpu.Flags.SetDecimal(v)
		cpu.Flags.SetOverflow(v)
		cpu.Flags.SetNegative(v)
		return 4, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x30: {Opcode: 0x30, Label: "BMI", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) (uint8, string) {
		cycles := uint8(2)
		offset, s := cpu.Fetch()
		cpu.TempAddress = cpu.GetPC() + uint16(offset)
		if cpu.Flags.GetFlag(gemu.Negative) {
			cycles += 1
			cpu.SetPC(cpu.TempAddress)
		}
		if PageCrossed(cpu.PrevPC, cpu.TempAddress) {
			cycles += 1
		}
		return cycles, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.TempAddress)
	}},
	0x09: {Opcode: 0x09, Label: "ORA", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		v, s := cpu.Fetch()
		r := v | cpu.A.GetValue()
		cpu.A.SetRegister(r)
		cpu.Flags.SetNegative(r)
		cpu.Flags.SetZeroByValue(r)
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0xB8: {Opcode: 0xB8, Label: "CLV", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		cpu.Flags.SetFlag(gemu.Overflow, false)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x49: {Opcode: 0x09, Label: "EOR", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		v, s := cpu.Fetch()
		r := v ^ cpu.A.GetValue()
		cpu.A.SetRegister(r)
		cpu.Flags.SetNegative(r)
		cpu.Flags.SetZeroByValue(r)
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0x69: {Opcode: 0x69, Label: "ADC", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		v, s := cpu.Fetch()
		r := uint16(v) + uint16(cpu.A.GetValue()) + uint16(cpu.Flags.GetFlagUint8(gemu.Carry))
		cf := false
		if r > 0xFF {
			r = 0 //r - 0xFF
			cf = true
		}
		r8 := uint8(r)

		cpu.Flags.SetFlag(gemu.Carry, cf)
		cpu.Flags.SetZeroByValue(r8)
		of := (r8 ^ cpu.A.GetValue()) & (r8 ^ v) & 0x80
		cpu.Flags.SetFlag(gemu.Overflow, of != 0)
		cpu.Flags.SetNegative(r8)
		cpu.A.SetRegister(r8)
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0xA0: {Opcode: 0xA0, Label: "LDY", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		v, s := cpu.Fetch()
		cpu.Y.SetRegister(v)
		cpu.Flags.SetZeroByValue(v)
		cpu.Flags.SetNegative(v)
		cpu.TempValue = v
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0xC0: {Opcode: 0xC0, Label: "CPY", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		v, s := cpu.Fetch()
		r := cpu.Y.GetValue() - v
		cpu.Flags.SetFlag(gemu.Carry, cpu.Y.GetValue() >= v)
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0xE0: {Opcode: 0xE0, Label: "CPX", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		v, s := cpu.Fetch()
		r := cpu.X.GetValue() - v
		cpu.Flags.SetFlag(gemu.Carry, cpu.X.GetValue() >= v)
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0xE9: {Opcode: 0xE9, Label: "SBC", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) (uint8, string) {
		v, s := cpu.Fetch()
		a := cpu.A.GetValue()
		c := cpu.Flags.GetFlagUint8(gemu.Carry)
		r := int8(a) + int8(^v) + int8(c)

		r8 := uint8(r)

		cpu.Flags.SetFlag(gemu.Zero, r == 0 && !cpu.Flags.GetFlag(gemu.Negative))

		of := (r8 ^ a) & (r8 ^ ^v) & 0x80
		cpu.Flags.SetFlag(gemu.Overflow, of != 0)

		cpu.Flags.SetNegative(r8)
		if cpu.Flags.GetFlag(gemu.Negative) {
			cpu.Flags.SetFlag(gemu.Carry, false)
		} else {
			cpu.Flags.SetFlag(gemu.Carry, true)
		}

		cpu.A.SetRegister(r8)
		return 2, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.TempAddress)
	}},
	0xC8: {Opcode: 0xC8, Label: "INY", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		// cpu.StackPush(cpu.A.GetValue())
		r := cpu.Y.GetValue() + 1
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		cpu.Y.SetRegister(r)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0xE8: {Opcode: 0xE8, Label: "INX", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		r := cpu.X.GetValue() + 1
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		cpu.X.SetRegister(r)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x88: {Opcode: 0x88, Label: "DEY", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		r := cpu.Y.GetValue() - 1
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		cpu.Y.SetRegister(r)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0xCA: {Opcode: 0xCA, Label: "DEX", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		r := cpu.X.GetValue() - 1
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		cpu.X.SetRegister(r)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0xA8: {Opcode: 0xA8, Label: "TAY", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		r := cpu.A.GetValue()
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		cpu.Y.SetRegister(r)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0xAA: {Opcode: 0xAA, Label: "TAX", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		r := cpu.A.GetValue()
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		cpu.X.SetRegister(r)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x98: {Opcode: 0x98, Label: "TYA", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		r := cpu.Y.GetValue()
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		cpu.A.SetRegister(r)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x8A: {Opcode: 0x8A, Label: "TXA", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		r := cpu.X.GetValue()
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		cpu.A.SetRegister(r)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0xBA: {Opcode: 0xBA, Label: "TSX", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		r := cpu.SP
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		cpu.X.SetRegister(r)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x8E: {Opcode: 0x8E, Label: "STX", Length: 3, AddressMode: cpu.Absolute, Function: func(cpu *cpu.CPU) (uint8, string) {
		ta, s := cpu.Fetch16() // uint16(cpu.Fetch())
		cpu.TempAddress = ta
		cpu.Store(cpu.TempAddress, cpu.X.GetValue())
		return 4, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X = %02X", cpu.TempAddress, cpu.X.GetPrevious())
	}},
	0x9A: {Opcode: 0x9A, Label: "TXS", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		r := cpu.X.GetValue()
		// cpu.Flags.SetZeroByValue(r)
		// cpu.Flags.SetNegative(r)
		cpu.SP = r
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0xAE: {Opcode: 0xAE, Label: "LDX", Length: 3, AddressMode: cpu.Absolute, Function: func(cpu *cpu.CPU) (uint8, string) {
		ta, s := cpu.Fetch16()
		cpu.TempAddress = ta
		v := cpu.FetchAddress(cpu.TempAddress)
		// cpu.X.SetRegister(cpu.Fetch())
		cpu.X.SetRegister(v)
		cpu.Flags.SetZeroByValue(cpu.X.GetValue())
		cpu.Flags.SetNegative(cpu.X.GetValue())
		return 4, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X = %02X", cpu.TempAddress, cpu.X.GetValue())
	}},
	0xAD: {Opcode: 0xAD, Label: "LDA", Length: 3, AddressMode: cpu.Absolute, Function: func(cpu *cpu.CPU) (uint8, string) {
		ta, s := cpu.Fetch16()
		cpu.TempAddress = ta
		v := cpu.FetchAddress(cpu.TempAddress) // - 0x0100)
		cpu.A.SetRegister(v)
		cpu.Flags.SetZeroByValue(v)
		cpu.Flags.SetNegative(v)
		return 4, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X = %02X", cpu.TempAddress, cpu.A.GetValue())
	}},
	0x40: {Opcode: 0x40, Label: "RTI", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) (uint8, string) {
		// pull NVxxDIZC flags from stack
		f := cpu.StackPop()
		cpu.Flags.SetCarry(f)
		cpu.Flags.SetZero(f)
		cpu.Flags.SetInterruptDisable(f)
		cpu.Flags.SetDecimal(f)
		cpu.Flags.SetOverflow(f)
		cpu.Flags.SetNegative(f)
		// pull PC from stack
		lo := cpu.StackPop()
		hi := cpu.StackPop()
		nsp := ToAddress(hi, lo)
		cpu.SetPC(nsp)

		return 6, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x4A: {Opcode: 0x4A, Label: "LSR", Length: 1, AddressMode: cpu.Accumulator, Function: func(cpu *cpu.CPU) (uint8, string) {
		// value = value >> 1, or visually: 0 -> [76543210] -> C
		a := cpu.A.GetValue()
		cpu.Flags.SetCarry(a)
		v := a >> 1
		cpu.A.SetRegister(v)
		cpu.Flags.SetZeroByValue(v)
		cpu.Flags.SetFlag(gemu.Negative, false)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return "A"
	}},
	0x0A: {Opcode: 0x0A, Label: "ASL", Length: 1, AddressMode: cpu.Accumulator, Function: func(cpu *cpu.CPU) (uint8, string) {
		// value = value >> 1, or visually: 0 -> [76543210] -> C
		a := cpu.A.GetValue()
		cpu.Flags.SetFlag(gemu.Carry, a&0x80 != 0)
		v := a << 1
		// cpu.Flags.SetCarry(v)
		cpu.A.SetRegister(v)
		cpu.Flags.SetZeroByValue(v)
		cpu.Flags.SetNegative(v)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return "A"
	}},
	0x6A: {Opcode: 0x6A, Label: "ROR", Length: 1, AddressMode: cpu.Accumulator, Function: func(cpu *cpu.CPU) (uint8, string) {
		// value = value >> 1 through C, or visually: C -> [76543210] -> C
		a := cpu.A.GetValue()
		v := a >> 1
		if cpu.Flags.GetFlag(gemu.Carry) {
			v = v | 0x80
		}
		cpu.Flags.SetFlag(gemu.Carry, a&0x01 != 0)
		// cpu.Flags.SetCarry(v)
		cpu.A.SetRegister(v)
		cpu.Flags.SetZeroByValue(v)
		cpu.Flags.SetNegative(v)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return "A"
	}},
	0x2A: {Opcode: 0x2A, Label: "ROL", Length: 1, AddressMode: cpu.Accumulator, Function: func(cpu *cpu.CPU) (uint8, string) {
		// value = value >> 1 through C, or visually: C -> [76543210] -> C
		a := cpu.A.GetValue()
		v := a << 1
		if cpu.Flags.GetFlag(gemu.Carry) {
			v = v | 0x01
		}
		cpu.Flags.SetFlag(gemu.Carry, a&0x80 != 0)
		// cpu.Flags.SetCarry(v)
		cpu.A.SetRegister(v)
		cpu.Flags.SetZeroByValue(v)
		cpu.Flags.SetNegative(v)
		return 2, ""
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return "A"
	}},
	0xA5: {Opcode: 0xA5, Label: "LDA", Length: 2, AddressMode: cpu.ZeroPageA, Function: func(cpu *cpu.CPU) (uint8, string) {
		ta, s := cpu.Fetch()
		// cpu.TempValue = ta
		cpu.TempValue = cpu.FetchAddress(uint16(ta) & 0x00FF)
		cpu.A.SetRegister(cpu.TempValue)
		cpu.Flags.SetZeroByValue(cpu.TempValue)
		cpu.Flags.SetNegative(cpu.TempValue)
		return 3, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%02X = %02X", cpu.TempAddress, cpu.A.GetValue())
	}},
	0x8D: {Opcode: 0x8D, Label: "STA", Length: 3, AddressMode: cpu.Absolute, Function: func(cpu *cpu.CPU) (uint8, string) {
		a, s := cpu.Fetch16()
		cpu.TempAddress = a
		cpu.TempValue = cpu.FetchAddress(cpu.TempAddress)
		cpu.Store(cpu.TempAddress, cpu.A.GetValue())
		return 4, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X = %02X", cpu.TempAddress, cpu.TempValue)
	}},
	0xA1: {Opcode: 0xA1, Label: "LDA", Length: 2, AddressMode: cpu.IndirectX, Function: func(cpu *cpu.CPU) (uint8, string) {
		// instruction declares the base
		base, s := cpu.Fetch()
		// now add the x
		zpa := base + cpu.X.GetValue()
		cpu.TempValue = zpa
		// lo is that byte
		lo := cpu.FetchAddress(uint16(zpa))
		// hi is next
		hi := cpu.FetchAddress(uint16(zpa + 1))
		// create the address
		ta := ToAddress(hi, lo)
		cpu.TempValue16 = ta

		// accumulator will be the val from this address
		a := cpu.FetchAddress(ta)
		cpu.A.SetRegister(a)

		cpu.Flags.SetZeroByValue(a)
		cpu.Flags.SetNegative(a)

		return 6, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("($%02X,X) @ %02X = %04X = %02X", cpu.TempAddress, cpu.TempValue, cpu.TempValue16, cpu.A.GetValue())
	}},
	0x81: {Opcode: 0xA1, Label: "STA", Length: 2, AddressMode: cpu.IndirectX, Function: func(cpu *cpu.CPU) (uint8, string) {
		// instruction declares the base
		base, s := cpu.Fetch()
		// now add the x
		zpa := base + cpu.X.GetValue()
		cpu.TempValue = zpa
		// lo is that byte
		lo := cpu.FetchAddress(uint16(zpa))
		// hi is next
		hi := cpu.FetchAddress(uint16(zpa + 1))
		// create the address
		ta := ToAddress(hi, lo)
		cpu.TempValue16 = ta

		cpu.TempAddressValue = cpu.FetchAddress(ta)

		cpu.Store(ta, cpu.A.GetValue())

		return 6, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("($%02X,X) @ %02X = %04X = %02X", cpu.TempAddress, cpu.TempValue, cpu.TempValue16, cpu.TempAddressValue)
	}},
	0x01: {Opcode: 0xA1, Label: "ORA", Length: 2, AddressMode: cpu.IndirectX, Function: func(cpu *cpu.CPU) (uint8, string) {
		// instruction declares the base
		base, s := cpu.Fetch()
		// now add the x
		zpa := base + cpu.X.GetValue()
		cpu.TempValue = zpa
		// lo is that byte
		lo := cpu.FetchAddress(uint16(zpa))
		// hi is next
		hi := cpu.FetchAddress(uint16(zpa + 1))
		// create the address
		ta := ToAddress(hi, lo)
		cpu.TempValue16 = ta

		cpu.TempAddressValue = cpu.FetchAddress(ta)

		a := cpu.A.GetValue()
		v := a | cpu.TempAddressValue
		cpu.A.SetRegister(v)

		cpu.Flags.SetZeroByValue(v)
		cpu.Flags.SetNegative(v)

		return 6, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("($%02X,X) @ %02X = %04X = %02X", cpu.TempAddress, cpu.TempValue, cpu.TempValue16, cpu.TempAddressValue)
	}},
	0x21: {Opcode: 0x21, Label: "AND", Length: 2, AddressMode: cpu.IndirectX, Function: func(cpu *cpu.CPU) (uint8, string) {
		// instruction declares the base
		base, s := cpu.Fetch()
		// now add the x
		zpa := base + cpu.X.GetValue()
		cpu.TempValue = zpa
		// lo is that byte
		lo := cpu.FetchAddress(uint16(zpa))
		// hi is next
		hi := cpu.FetchAddress(uint16(zpa + 1))
		// create the address
		ta := ToAddress(hi, lo)
		cpu.TempValue16 = ta

		cpu.TempAddressValue = cpu.FetchAddress(ta)

		a := cpu.A.GetValue()
		v := a & cpu.TempAddressValue
		cpu.A.SetRegister(v)

		cpu.Flags.SetZeroByValue(v)
		cpu.Flags.SetNegative(v)

		return 6, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("($%02X,X) @ %02X = %04X = %02X", cpu.TempAddress, cpu.TempValue, cpu.TempValue16, cpu.TempAddressValue)
	}},
	0x41: {Opcode: 0x41, Label: "EOR", Length: 2, AddressMode: cpu.IndirectX, Function: func(cpu *cpu.CPU) (uint8, string) {
		// instruction declares the base
		base, s := cpu.Fetch()
		// now add the x
		zpa := base + cpu.X.GetValue()
		cpu.TempValue = zpa
		// lo is that byte
		lo := cpu.FetchAddress(uint16(zpa))
		// hi is next
		hi := cpu.FetchAddress(uint16(zpa + 1))
		// create the address
		ta := ToAddress(hi, lo)
		cpu.TempValue16 = ta

		cpu.TempAddressValue = cpu.FetchAddress(ta)

		a := cpu.A.GetValue()
		v := a ^ cpu.TempAddressValue
		cpu.A.SetRegister(v)

		cpu.Flags.SetZeroByValue(v)
		cpu.Flags.SetNegative(v)

		return 6, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("($%02X,X) @ %02X = %04X = %02X", cpu.TempAddress, cpu.TempValue, cpu.TempValue16, cpu.TempAddressValue)
	}},
	0x61: {Opcode: 0x61, Label: "ADC", Length: 2, AddressMode: cpu.IndirectX, Function: func(cpu *cpu.CPU) (uint8, string) {
		// instruction declares the base
		base, s := cpu.Fetch()
		// now add the x
		zpa := base + cpu.X.GetValue()
		cpu.TempValue = zpa
		// lo is that byte
		lo := cpu.FetchAddress(uint16(zpa))
		// hi is next
		hi := cpu.FetchAddress(uint16(zpa + 1))
		// create the address
		ta := ToAddress(hi, lo)
		cpu.TempValue16 = ta
		cpu.TempAddressValue = cpu.FetchAddress(ta)

		r := uint16(cpu.TempAddressValue) + uint16(cpu.A.GetValue()) + uint16(cpu.Flags.GetFlagUint8(gemu.Carry))
		cf := false
		if r > 0xFF {
			r = 0 //r - 0xFF
			cf = true
		}
		r8 := uint8(r)

		cpu.Flags.SetFlag(gemu.Carry, cf)
		cpu.Flags.SetZeroByValue(r8)
		of := (r8 ^ cpu.A.GetValue()) & (r8 ^ cpu.TempAddressValue) & 0x80
		cpu.Flags.SetFlag(gemu.Overflow, of != 0)
		cpu.Flags.SetNegative(r8)
		cpu.A.SetRegister(r8)

		return 6, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("($%02X,X) @ %02X = %04X = %02X", cpu.TempAddress, cpu.TempValue, cpu.TempValue16, cpu.TempAddressValue)
	}},
	0xC1: {Opcode: 0xC1, Label: "CMP", Length: 2, AddressMode: cpu.IndirectX, Function: func(cpu *cpu.CPU) (uint8, string) {
		// instruction declares the base
		base, s := cpu.Fetch()
		// now add the x
		zpa := base + cpu.X.GetValue()
		cpu.TempValue = zpa
		// lo is that byte
		lo := cpu.FetchAddress(uint16(zpa))
		// hi is next
		hi := cpu.FetchAddress(uint16(zpa + 1))
		// create the address
		ta := ToAddress(hi, lo)
		cpu.TempValue16 = ta
		cpu.TempAddressValue = cpu.FetchAddress(ta)

		a := cpu.A.GetValue()
		v := cpu.TempAddressValue
		r := a - v

		cpu.Flags.SetFlag(gemu.Carry, a >= v)
		cpu.Flags.SetFlag(gemu.Zero, a == v)
		cpu.Flags.SetNegative(r)

		return 6, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("($%02X,X) @ %02X = %04X = %02X", cpu.TempAddress, cpu.TempValue, cpu.TempValue16, cpu.TempAddressValue)
	}},
	0xE1: {Opcode: 0xE1, Label: "SBC", Length: 2, AddressMode: cpu.IndirectX, Function: func(cpu *cpu.CPU) (uint8, string) {
		// instruction declares the base
		base, s := cpu.Fetch()
		// now add the x
		zpa := base + cpu.X.GetValue()
		cpu.TempValue = zpa
		// lo is that byte
		lo := cpu.FetchAddress(uint16(zpa))
		// hi is next
		hi := cpu.FetchAddress(uint16(zpa + 1))
		// create the address
		ta := ToAddress(hi, lo)
		cpu.TempValue16 = ta
		cpu.TempAddressValue = cpu.FetchAddress(ta)

		a := cpu.A.GetValue()
		v := cpu.TempAddressValue
		c := cpu.Flags.GetFlagUint8(gemu.Carry)
		r := int8(a) + int8(^v) + int8(c)

		r8 := uint8(r)

		cpu.Flags.SetFlag(gemu.Zero, r == 0 && !cpu.Flags.GetFlag(gemu.Negative))

		of := (r8 ^ a) & (r8 ^ ^v) & 0x80
		cpu.Flags.SetFlag(gemu.Overflow, of != 0)

		cpu.Flags.SetNegative(r8)
		if cpu.Flags.GetFlag(gemu.Negative) {
			cpu.Flags.SetFlag(gemu.Carry, false)
		} else {
			cpu.Flags.SetFlag(gemu.Carry, true)
		}

		cpu.A.SetRegister(r8)

		return 6, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("($%02X,X) @ %02X = %04X = %02X", cpu.TempAddress, cpu.TempValue, cpu.TempValue16, cpu.TempAddressValue)
	}},
	0xA4: {Opcode: 0xA4, Label: "LDY", Length: 2, AddressMode: cpu.ZeroPageA, Function: func(cpu *cpu.CPU) (uint8, string) {
		ta, s := cpu.Fetch()
		v := cpu.FetchAddress(uint16(ta))
		cpu.Y.SetRegister(v)
		cpu.Flags.SetZeroByValue(cpu.Y.GetValue())
		cpu.Flags.SetNegative(cpu.Y.GetValue())
		return 3, s
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%02X = %02X", cpu.TempAddress, cpu.Y.GetValue())
	}},
}

func ToAddress(hi uint8, lo uint8) uint16 {
	return (uint16(hi) << 8) | uint16(lo)
}

var counter uint64 = 0

func main() {
	stopAfter := -1
	if len(os.Args) > 1 {
		stopAfterStr := os.Args[1]
		if len(stopAfterStr) > 0 {
			val, err := strconv.Atoi(stopAfterStr)
			if err != nil {
				log.Panic("Invalid param")
			}
			stopAfter = val
		}
	}

	rom := gemu.Cartridge{}
	err := rom.Insert("nestest.nes")
	if err != nil {
		fmt.Println("Error inserting ROM:", err)
		return
	}
	fmt.Println("ROM inserted successfully")

	cpu := cpu.CPU{}
	cpu.Reset()
	cpu.LoadCartridge(rom)
	cpu.SetPC(0xC000)

	ref, err := os.Open("./reference.txt")
	if err != nil {
		fmt.Println("Error opening reference file:", err)
		return
	}
	defer ref.Close()
	refScanner := bufio.NewScanner(ref)

	for {
		if cpu.CyclesRemaining == 0 {
			var refLine string
			if refScanner.Scan() {
				refLine = refScanner.Text()
			} else {
				fmt.Println("No more lines in the reference file")
				return
			}

			var line string
			counter += 1
			// print the counter (not part of the reference)
			fmt.Printf("%4d  ", counter)
			// print the current PC
			line += fmt.Sprintf("%04X  ", cpu.GetPC())

			// fetch instruction
			opcode, os := cpu.Fetch()
			line += os

			// decode instruction
			instruction, ok := instructions[opcode]
			if !ok {
				fmt.Printf("Unknown opcode: %02X\n", opcode)
				break
			}

			// generate the current state
			state := cpu.PrintDetails(instruction.AddressMode)

			// execute instruction
			cr, is := instruction.Function(&cpu)
			cpu.CyclesRemaining = cr
			line += is

			makeup := 3 * (3 - instruction.Length)
			if makeup > 0 {
				line += fmt.Sprint(strings.Repeat(" ", makeup+1))
			}
			line += fmt.Sprintf("%s %-27s ", instruction.Label, instruction.PrintDetails(cpu, instruction))

			// print details
			// line += fmt.Sprint(state)
			line += state

			// actually print
			fmt.Println(line)

			if line != refLine {
				fmt.Println("No match")
				fmt.Println(line)
				fmt.Println("VV REF VV")
				fmt.Println(refLine)
				break
			}

			// if counter == 878 {
			// 	cpu.PrintStack()
			// }

			if counter == uint64(stopAfter) {
				break
			}
		}

		cpu.TotalCycles++
		cpu.CyclesRemaining--
	}
}
