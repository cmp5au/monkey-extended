package jit

import (
	"fmt"
	"syscall"

	"github.com/cmp5au/monkey-extended/code"
	"github.com/cmp5au/monkey-extended/object"
)

func JitCompileFunctions(constants []object.Object) {
	fmt.Println("On darwin_arm64")
	// if we imagine program's call structure as a trie, the functions are listed
	// in Bytecode.Constants in postfix order: the same order we'd like to JIT compile
	for _, c := range constants {
		if fn, ok := c.(*object.CompiledFunction); ok {
			mustJitCompile(fn)
		}
	}
}

func mustJitCompile(fn *object.CompiledFunction) {
	// 1. Compile function bytecode to machine code; early return if failure
	machineCodeBuf := compileInstructions(fn.Instructions)
	if len(machineCodeBuf) == 0 {
		return
	}

	// 2. Allocate memory for machine code via mmap. At this point, the memory is not executable, but read-writable.
	execBuf := mustAllocateByMMap(len(machineCodeBuf))

	// 3. Write machine code to machineCodeBuf.
	copy(execBuf, machineCodeBuf)

	// 4. Mark the memory region as executable. This marks the memory region as read-executable.
	mustMarkAsExecutable(machineCodeBuf)

	// 5. Write the machine code back to the CompiledFunction.
	go func(machineCodeBuf []byte) {
		defer fn.JitInstructions.Unlock()
		fn.JitInstructions.Lock()
		fn.JitInstructions.MachineCodeInstructions = machineCodeBuf
	}(machineCodeBuf)
}

// mustAllocateByMMap returns a memory region that is read-writable via mmap.
func mustAllocateByMMap(size int) []byte {
	machineCodes, err := syscall.Mmap(-1, 0,
		size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_ANON|syscall.MAP_PRIVATE,
	)
	if err != nil {
		panic(err)
	}
	return machineCodes
}

// mustMarkAsExecutable marks the memory region as read-executable via mprotect.
func mustMarkAsExecutable(machineCodes []byte) {
	if err := syscall.Mprotect(
		machineCodes,
		syscall.PROT_READ|syscall.PROT_EXEC,
	); err != nil {
		panic(err)
	}
}

func compileInstructions(ins code.Instructions) []byte {
	return nil
}
