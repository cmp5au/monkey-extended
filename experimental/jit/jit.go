//go:build !darwin || !arm64
package jit

import (
	"fmt"

	"github.com/cmp5au/monkey-extended/object"
)

func JitCompileFunctions(constants []object.Object) {
	fmt.Println("Not on darwin_arm64")
	return
}
