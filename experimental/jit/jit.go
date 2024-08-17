//go:build !darwin || !arm64

package jit

import (
	"github.com/cmp5au/monkey-extended/object"
)

func JitCompileFunctions(constants []object.Object) {}

func ExecMem(cl *object.Closure, sp *int) {}

func execMemInner(mem []byte, sp uintptr) {}
