package jit

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestMoveWideVal(t *testing.T) {
	val := uintptr(binary.LittleEndian.Uint64([]byte{0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 0x01}))
	fmt.Printf("%x\n", val)
	x0Mov := movWideVal(val, 0)
	expected := []byte{
		0xe0, 0xbd, 0x99, 0xd2,
		0x60, 0x35, 0xb1, 0xf2,
		0xe0, 0xac, 0xc8, 0xf2,
		0x60, 0x24, 0xe0, 0xf2,
	}
	if len(x0Mov) != len(expected) {
		t.Errorf("Failed to correctly encode move instructions for 0x0123456789abcdef into register 0. Got %d instructions", len(x0Mov))
	}
	for i := range x0Mov {
		if x0Mov[i] != expected[i] {
			t.Errorf("Failed to correctly encode move instructions for 0x0123456789abcdef into register 0. expected=%x, got=%x", expected[i], x0Mov[i])
		}
	}
}
