package jit

import (
	"syscall"

	"github.com/cmp5au/monkey-extended/code"
	"github.com/cmp5au/monkey-extended/object"
)

func JitCompileFunctions(constants []object.Object) {
	// if we imagine program's call structure as a trie, the functions are listed
	// in Bytecode.Constants in postfix order: the same order we'd like to JIT compile
	for _, c := range constants {
		if fn, ok := c.(*object.CompiledFunction); ok {
			mustJitCompile(fn)
		}
	}
}

// add defer recover to harden this
func mustJitCompile(fn *object.CompiledFunction) {
	machineCodeBuf := compileInstructions(fn.Instructions)
	if len(machineCodeBuf) == 0 {
		return
	}

	execBuf := mustAllocateByMMap(len(machineCodeBuf))

	copy(execBuf, machineCodeBuf)

	mustMarkAsExecutable(machineCodeBuf)

	go func(machineCodeBuf []byte) {
		defer fn.JitInstructions.Unlock()
		fn.JitInstructions.Lock()
		fn.JitInstructions.MachineCodeInstructions = machineCodeBuf
	}(machineCodeBuf)
}

func mustAllocateByMMap(size int) []byte {
	mem, err := syscall.Mmap(
		-1,
		0,
		size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_ANON|syscall.MAP_PRIVATE,
	)
	if err != nil {
		panic(err)
	}
	return mem
}

func mustMarkAsExecutable(mem []byte) {
	if err := syscall.Mprotect(
		mem,
		syscall.PROT_READ|syscall.PROT_EXEC,
	); err != nil {
		panic(err)
	}
}

func compileInstructions(ins code.Instructions) []byte {
	return nil
}

func ExecMem(mem []byte, sp int)
