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

	Temp   uint16
	PrevPC uint16

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

func (cpu *CPU) Fetch() uint8 {
	cpu.Temp = uint16(0x0)<<8 | uint16(cpu.memory[cpu.pc])
	fmt.Printf("%02X ", cpu.Temp)
	cpu.pc++
	return uint8(cpu.Temp & 0xFF)
}

func (cpu *CPU) Fetch16() uint16 {
	low := cpu.Fetch()
	high := cpu.Fetch()
	fmt.Print(" ")
	cpu.Temp = uint16(high)<<8 | uint16(low)
	return cpu.Temp
}

func (cpu *CPU) FetchAddress(addr uint16) uint8 {
	return cpu.memory[addr]
}

func (cpu *CPU) Store(addr uint16, v uint8) {
	cpu.memory[addr] = v
}

func (cpu *CPU) StackPush(v uint8) {
	cpu.memory[cpu.SP] = v
	cpu.SP--
}

func (cpu *CPU) StackPop() uint8 {
	cpu.SP++
	r := cpu.memory[cpu.SP]
	return r
}

// const for address modes
const (
	Absolute = iota
	AbsoluteX
	AbsoluteY
	Immediate
	ZeroPageA
	ZeroPageX
	ZeroPageY
	Implicit
	Relative
)

func (cpu CPU) PrintDetails(addressMode uint8) string {

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
