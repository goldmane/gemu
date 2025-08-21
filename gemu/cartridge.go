package gemu

import (
	"fmt"
	"os"
)

type Cartridge struct {
	Header  [16]byte
	Trainer []byte // 512 bytes
	PRG     []byte // 16kb units
	CHR     []byte // 8kb units
}

func (c *Cartridge) Insert(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	bytesRead, err := file.Read(c.Header[:])
	if err != nil {
		return err
	}

	if bytesRead != len(c.Header) {
		return fmt.Errorf("failed to read header")
	}

	// validate the header
	if c.Header[0] != 0x4E || c.Header[1] != 0x45 || c.Header[2] != 0x53 || c.Header[3] != 0x1A {
		return fmt.Errorf("invalid header")
	}

	c.PRG = make([]byte, uint(c.Header[4])*16384)
	fmt.Printf("Byte 4 (PRG): %d * 16kb units (%d total)\n", c.Header[4], len(c.PRG))
	bytesRead, err = file.Read(c.PRG)
	if err != nil {
		return err
	}
	if bytesRead != len(c.PRG) {
		return fmt.Errorf("failed to read PRG")
	}

	if c.Header[5] == 0 {
		fmt.Println("Byte 5 (CHR RAM)")
	} else {
		c.CHR = make([]byte, uint(c.Header[5])*8192)
		fmt.Printf("Byte 5 (CHR ROM): %d * 8kb units (%d total)\n", c.Header[5], len(c.CHR))
		bytesRead, err = file.Read(c.CHR)
		if err != nil {
			return err
		}
		if bytesRead != len(c.CHR) {
			return fmt.Errorf("failed to read CHR")
		}
	}

	// print byte 6 in binary
	fmt.Printf("Byte 6 (Flags 6): %08b\n", c.Header[6])
	fmt.Printf("Byte 7 (Flags 7): %08b\n", c.Header[7])

	return nil
}
