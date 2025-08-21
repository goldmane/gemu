package main

import (
	"fmt"
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
	Function     func(cpu *cpu.CPU) uint8
	PrintDetails func(cpu cpu.CPU, ins Instruction) string
}

var instructions = map[uint8]Instruction{
	0x4C: {Opcode: 0x4C, Label: "JMP", Length: 3, AddressMode: cpu.Absolute, Function: func(cpu *cpu.CPU) uint8 {
		cpu.Temp = cpu.Fetch16()
		cpu.SetPC(cpu.Temp)
		return 3
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.Temp)
	}},
	0xA2: {Opcode: 0xA2, Label: "LDX", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) uint8 {
		cpu.X.SetRegister(cpu.Fetch())
		cpu.Flags.SetZeroByValue(cpu.X.GetValue())
		cpu.Flags.SetNegative(cpu.X.GetValue())
		return 2
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.Temp)
	}},
	0x86: {Opcode: 0x86, Label: "STX", Length: 2, AddressMode: cpu.ZeroPageX, Function: func(cpu *cpu.CPU) uint8 {
		cpu.Temp = uint16(cpu.Fetch())
		cpu.Store(cpu.Temp, cpu.X.GetValue())
		return 3
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%02X = %02X", cpu.Temp, cpu.X.GetValue())
	}},
	0x20: {Opcode: 0x86, Label: "JSR", Length: 3, AddressMode: cpu.Absolute, Function: func(cpu *cpu.CPU) uint8 {
		// push the current PC + 2 onto the stack
		cpu.StackPush(HighByte(cpu.GetPC() + 2))
		cpu.StackPush(LowByte(cpu.GetPC() + 2))
		// get the target address
		cpu.Temp = cpu.Fetch16()
		// go to target
		cpu.SetPC(cpu.Temp)
		return 6
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.Temp)
	}},
	0xEA: {Opcode: 0x86, Label: "NOP", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		// nothing to do here
		return 2
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x38: {Opcode: 0xA2, Label: "SEC", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		cpu.Flags.SetFlag(gemu.Carry, true)
		return 2
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0xB0: {Opcode: 0xA2, Label: "BCS", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) uint8 {
		cycles := uint8(2)
		offset := cpu.Fetch()
		cpu.Temp = cpu.GetPC() + uint16(offset)
		if cpu.Flags.GetFlag(gemu.Carry) {
			cycles += 1
			cpu.SetPC(cpu.Temp)
		}
		if PageCrossed(cpu.PrevPC, cpu.Temp) {
			cycles += 1
		}
		return cycles
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.Temp)
	}},
	0x18: {Opcode: 0xA2, Label: "CLC", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		cpu.Flags.SetFlag(gemu.Carry, false)
		return 2
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x90: {Opcode: 0xA2, Label: "BCC", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) uint8 {
		cycles := uint8(2)
		offset := cpu.Fetch()
		cpu.Temp = cpu.GetPC() + uint16(offset)
		if !cpu.Flags.GetFlag(gemu.Carry) {
			cycles += 1
			cpu.SetPC(cpu.Temp)
		}
		if PageCrossed(cpu.PrevPC, cpu.Temp) {
			cycles += 1
		}
		return cycles
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.Temp)
	}},
	0xA9: {Opcode: 0xA2, Label: "LDA", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) uint8 {
		tmp := cpu.Fetch()
		cpu.A.SetRegister(tmp)
		cpu.Flags.SetZeroByValue(tmp)
		cpu.Flags.SetNegative(tmp)
		return 2
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.Temp)
	}},
	0xF0: {Opcode: 0xA2, Label: "BEQ", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) uint8 {
		cycles := uint8(2)
		offset := cpu.Fetch()
		cpu.Temp = cpu.GetPC() + uint16(offset)
		if cpu.Flags.GetFlag(gemu.Zero) {
			cycles += 1
			cpu.SetPC(cpu.Temp)
		}
		if PageCrossed(cpu.PrevPC, cpu.Temp) {
			cycles += 1
		}
		return cycles
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.Temp)
	}},
	0xD0: {Opcode: 0xA2, Label: "BNE", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) uint8 {
		cycles := uint8(2)
		offset := cpu.Fetch()
		cpu.Temp = cpu.GetPC() + uint16(offset)
		if !cpu.Flags.GetFlag(gemu.Zero) {
			cycles += 1
			cpu.SetPC(cpu.Temp)
		}
		if PageCrossed(cpu.PrevPC, cpu.Temp) {
			cycles += 1
		}
		return cycles
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.Temp)
	}},
	0x85: {Opcode: 0x86, Label: "STA", Length: 2, AddressMode: cpu.ZeroPageA, Function: func(cpu *cpu.CPU) uint8 {
		a := cpu.Fetch()
		cpu.Temp = uint16(a)
		cpu.Store(cpu.Temp, cpu.A.GetValue())
		return 3
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%02X = %02X", cpu.Temp, cpu.A.GetPrevious())
	}},
	0x24: {Opcode: 0x86, Label: "BIT", Length: 2, AddressMode: cpu.ZeroPageA, Function: func(cpu *cpu.CPU) uint8 {
		a := cpu.Fetch()                 // get the address
		v := cpu.FetchAddress(uint16(a)) // get the value from that address
		cpu.Temp = uint16(a)
		r := v & cpu.A.GetValue()
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetOverflow(r)
		cpu.Flags.SetNegative(r)
		return 3
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%02X = %02X", cpu.Temp, cpu.A.GetValue())
	}},
	0x70: {Opcode: 0xA2, Label: "BVS", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) uint8 {
		cycles := uint8(2)
		offset := cpu.Fetch()
		cpu.Temp = cpu.GetPC() + uint16(offset)
		if cpu.Flags.GetFlag(gemu.Overflow) {
			cycles += 1
			cpu.SetPC(cpu.Temp)
		}
		if PageCrossed(cpu.PrevPC, cpu.Temp) {
			cycles += 1
		}
		return cycles
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.Temp)
	}},
	0x50: {Opcode: 0xA2, Label: "BVC", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) uint8 {
		cycles := uint8(2)
		offset := cpu.Fetch()
		cpu.Temp = cpu.GetPC() + uint16(offset)
		if !cpu.Flags.GetFlag(gemu.Overflow) {
			cycles += 1
			cpu.SetPC(cpu.Temp)
		}
		if PageCrossed(cpu.PrevPC, cpu.Temp) {
			cycles += 1
		}
		return cycles
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.Temp)
	}},
	0x10: {Opcode: 0xA2, Label: "BPL", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) uint8 {
		cycles := uint8(2)
		offset := cpu.Fetch()
		cpu.Temp = cpu.GetPC() + uint16(offset)
		if !cpu.Flags.GetFlag(gemu.Negative) {
			cycles += 1
			cpu.SetPC(cpu.Temp)
		}
		if PageCrossed(cpu.PrevPC, cpu.Temp) {
			cycles += 1
		}
		return cycles
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.Temp)
	}},
	0x60: {Opcode: 0x60, Label: "RTS", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		lo := cpu.StackPop()
		hi := cpu.StackPop()
		// fmt.Printf("\n\n%02x %02x\n\n", lo, hi)
		cpu.SetPC(ToAddress(hi, lo))
		return 6
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x78: {Opcode: 0x60, Label: "SEI", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		cpu.Flags.SetFlag(gemu.InterruptDisable, true)
		return 2
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0xF8: {Opcode: 0x60, Label: "SED", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		cpu.Flags.SetFlag(gemu.Decimal, true)
		return 2
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x08: {Opcode: 0x60, Label: "PHP", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		cpu.StackPush(cpu.Flags.Value())
		return 3
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x68: {Opcode: 0x68, Label: "PLA", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		v := cpu.StackPop()
		cpu.A.SetRegister(v + 0x10)
		cpu.Flags.SetNegative(v)
		cpu.Flags.SetZeroByValue(v)
		return 4
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x29: {Opcode: 0x26, Label: "AND", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) uint8 {
		v := cpu.Fetch()
		r := v & cpu.A.GetValue()
		cpu.A.SetRegister(r)
		cpu.Flags.SetNegative(r)
		cpu.Flags.SetZeroByValue(r)
		return 2
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.Temp)
	}},
	0xC9: {Opcode: 0xC9, Label: "CMP", Length: 2, AddressMode: cpu.Immediate, Function: func(cpu *cpu.CPU) uint8 {
		v := cpu.Fetch()
		r := cpu.A.GetValue() - v
		cpu.Flags.SetFlag(gemu.Carry, cpu.A.GetValue() >= v)
		cpu.Flags.SetZeroByValue(r)
		cpu.Flags.SetNegative(r)
		return 2
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("#$%02X", cpu.Temp)
	}},
	0xD8: {Opcode: 0xD8, Label: "CLD", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		cpu.Flags.SetFlag(gemu.Decimal, false)
		return 2
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x48: {Opcode: 0x48, Label: "PHA", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		cpu.StackPush(cpu.A.GetValue())
		return 3
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x28: {Opcode: 0x28, Label: "PLP", Length: 1, AddressMode: cpu.Implicit, Function: func(cpu *cpu.CPU) uint8 {
		v := cpu.StackPop()
		cpu.Flags.SetCarry(v)
		cpu.Flags.SetZero(v)
		cpu.Flags.SetInterruptDisable(v)
		cpu.Flags.SetDecimal(v)
		cpu.Flags.SetOverflow(v)
		cpu.Flags.SetNegative(v)
		return 4
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return ""
	}},
	0x30: {Opcode: 0x30, Label: "BMI", Length: 2, AddressMode: cpu.Relative, Function: func(cpu *cpu.CPU) uint8 {
		cycles := uint8(2)
		offset := cpu.Fetch()
		cpu.Temp = cpu.GetPC() + uint16(offset)
		if cpu.Flags.GetFlag(gemu.Negative) {
			cycles += 1
			cpu.SetPC(cpu.Temp)
		}
		if PageCrossed(cpu.PrevPC, cpu.Temp) {
			cycles += 1
		}
		return cycles
	}, PrintDetails: func(cpu cpu.CPU, ins Instruction) string {
		return fmt.Sprintf("$%04X", cpu.Temp)
	}},
}

func ToAddress(hi uint8, lo uint8) uint16 {
	return (uint16(hi) << 8) | uint16(lo)
}

func main() {
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

	counter := 0

	for {
		if cpu.CyclesRemaining == 0 {
			counter += 1
			// print the current PC
			fmt.Printf("%4d  %04X  ", counter, cpu.GetPC())

			// fetch instruction
			opcode := cpu.Fetch()

			// decode instruction
			instruction, ok := instructions[opcode]
			if !ok {
				fmt.Printf("Unknown opcode: %02X\n", opcode)
				break
			}

			// generate the current state
			state := cpu.PrintDetails(instruction.AddressMode)

			// execute instruction
			// set the cycles remaining
			cpu.CyclesRemaining = instruction.Function(&cpu)

			makeup := 3 * (3 - instruction.Length)
			if makeup > 0 {
				fmt.Print(strings.Repeat(" ", makeup+1))
			}
			fmt.Printf("%s %-28s ", instruction.Label, instruction.PrintDetails(cpu, instruction))

			// print details
			fmt.Println(state)
		}

		cpu.TotalCycles++
		cpu.CyclesRemaining--
	}
}
