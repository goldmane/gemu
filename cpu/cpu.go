package cpu

import (
	"fmt"

	"github.com/goldmane/gemu/gemu"
)

type RegisterSize interface {
	uint8 | uint16
}

type Register struct {
	value    uint8
	previous uint8
}

func (r *Register) SetRegister(v uint8) {
	r.previous = r.value
	r.value = v
}

func (r *Register) GetValue() uint8 {
	return r.value
}
func (r *Register) GetPrevious() uint8 {
	return r.previous
}

type Register16 struct {
	value    uint16
	previous uint16
}

func (r *Register16) SetRegister(v uint16) {
	r.previous = r.value
	r.value = v
}

type CPU struct {
	pc uint16
	SP uint8
	A  Register
	X  Register
	Y  Register

	Flags gemu.CpuFlag

	TempValue        uint8
	TempValue16      uint16
	TempAddress      uint16
	TempAddress_2    uint16
	TempAddressValue uint8
	PrevPC           uint16

	DetailsOverride string

	CyclesRemaining uint8
	TotalCycles     uint64

	memory []byte
}

func (cpu *CPU) Reset() {
	cpu.pc = 0xC000
	cpu.SP = 0xFD
	cpu.A = Register{value: 0x00, previous: 0x00}
	cpu.X = Register{value: 0x00, previous: 0x00}
	cpu.Y = Register{value: 0x00, previous: 0x00}

	cpu.TotalCycles = 7 // starting value

	// init the memory
	cpu.memory = make([]byte, 64*1024)

	// init the flags
	cpu.Flags.Reset()
}

func (cpu *CPU) LoadCartridge(c gemu.Cartridge) {
	copy(cpu.memory[0x8000:], c.PRG)
	copy(cpu.memory[0xC000:], c.PRG)
}

func (cpu *CPU) SetPC(v uint16) {
	cpu.PrevPC = cpu.pc
	cpu.pc = v
}

func (cpu *CPU) GetPC() uint16 {
	return cpu.pc
}

func (cpu *CPU) Fetch() (uint8, string) {
	cpu.TempAddress = uint16(0x0)<<8 | uint16(cpu.memory[cpu.pc])
	p := fmt.Sprintf("%02X ", cpu.TempAddress)
	cpu.PrevPC = cpu.pc
	cpu.pc++
	return uint8(cpu.TempAddress & 0xFF), p
}

func (cpu *CPU) Fetch16() (uint16, string) {
	low, ls := cpu.Fetch()
	high, hs := cpu.Fetch()
	cpu.TempAddress = uint16(high)<<8 | uint16(low)
	return cpu.TempAddress, (ls + hs + " ")
}

func (cpu *CPU) FetchAddress(addr uint16) uint8 {
	return cpu.memory[addr]
}

func (cpu *CPU) Store(addr uint16, v uint8) {
	cpu.memory[addr] = v
}

func (cpu *CPU) StackPush(v uint8) {
	a := uint16(0x0100) | uint16(cpu.SP)
	// cpu.memory[cpu.SP] = v
	cpu.memory[a] = v
	cpu.SP--
}

func (cpu *CPU) StackPop() uint8 {
	cpu.SP++
	a := uint16(0x0100) | uint16(cpu.SP)
	// r := cpu.memory[cpu.SP]
	r := cpu.memory[a]
	return r
}

// const for address modes
const (
	Absolute = iota
	AbsoluteX
	AbsoluteY
	Immediate
	ZeroPage
	ZeroPageX
	ZeroPageY
	Implicit
	Relative
	Accumulator
	IndirectX
	IndirectY
)

func (cpu CPU) PrintDetails(addressMode uint8, counter uint64) string {

	r1 := (func(addressMode uint8) string {
		var a, x, y uint8

		a = cpu.A.GetValue()
		x = cpu.X.GetValue()
		y = cpu.Y.GetValue()

		return fmt.Sprintf("A:%02X X:%02X Y:%02X", a, x, y)
	})(addressMode)

	// figure the ppu values
	t3 := cpu.TotalCycles * 3
	ppu1 := t3 / 341
	ppu2 := t3 % 341

	// print registers
	b := fmt.Sprintf("P:%02X SP:%02X PPU:%3d,%3d", cpu.Flags.Value(), cpu.SP, ppu1, ppu2)
	c := fmt.Sprintf("CYC:%d", cpu.TotalCycles)

	// return fmt.Sprintf("%-28s%s %s %s", d, r1, b, c)
	return fmt.Sprintf("%s %s %s", r1, b, c)
}

func (cpu CPU) GetMemory() []byte {
	return cpu.memory
}

func (cpu CPU) FindInMemory(v uint8) {
	fmt.Printf("\nLooking for %02X:\n", v)
	for i := 0; i < len(cpu.memory); i++ {
		if cpu.memory[i] == v {
			fmt.Printf("%04X\n", i)
		}
	}
}

func (cpu CPU) PrintStack() {
	start := uint16(0x01FD)
	end := (uint16(0x0100) | uint16(cpu.SP)) - 1
	fmt.Printf("\nStack from 0x01FD to 0x%04X:\n", end)
	for i := start; i >= end; i -= 0x01 {
		fmt.Printf("0x%04X: 0x%02X\n", i, cpu.memory[i])
	}
	fmt.Println()
}
