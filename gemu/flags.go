package gemu

const (
	Carry            = uint8(1 << 0)
	Zero             = byte(1 << 1)
	InterruptDisable = byte(1 << 2)
	Decimal          = byte(1 << 3)
	Break            = byte(1 << 4)
	Unused           = byte(1 << 5)
	Overflow         = byte(1 << 6)
	Negative         = byte(1 << 7)
)

type CpuFlag struct {
	flags byte
}

func (f *CpuFlag) Value() byte {
	return f.flags
}

func (f *CpuFlag) Reset() {
	f.flags = 0x24
}

func (f *CpuFlag) SetFlag(flag uint8, value bool) {
	if value {
		f.flags |= flag
	} else {
		f.flags &^= flag
	}
}

func (f *CpuFlag) GetFlag(flag uint8) bool {
	return (f.flags & flag) != 0
}

func (f *CpuFlag) GetFlagUint8(flag uint8) uint8 {
	if (f.flags & flag) != 0 {
		return 1
	} else {
		return 0
	}
}

func (f *CpuFlag) SetCarry(value uint8) {
	f.SetFlag(Carry, value&Carry != 0)
}

func (f *CpuFlag) SetZero(value uint8) {
	f.SetFlag(Zero, value&Zero != 0)
}

func (f *CpuFlag) SetZeroByValue(value uint8) {
	f.SetFlag(Zero, value == 0)
}

func (f *CpuFlag) SetInterruptDisable(value uint8) {
	f.SetFlag(InterruptDisable, value&InterruptDisable != 0)
}

func (f *CpuFlag) SetDecimal(value uint8) {
	f.SetFlag(Decimal, value&Decimal != 0)
}

func (f *CpuFlag) SetOverflow(value uint8) {
	f.SetFlag(Overflow, value&0x40 != 0)
}

func (f *CpuFlag) SetNegative(value uint8) {
	f.SetFlag(Negative, value&0x80 != 0)
}
