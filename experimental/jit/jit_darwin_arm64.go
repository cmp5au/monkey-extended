package jit

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/cmp5au/monkey-extended/object"
)

func JitCompileFunctions(constants []object.Object, done chan struct{}) {
	// if we imagine program's call structure as a trie, the functions are listed
	// in Bytecode.Constants in postfix order: the same order we'd like to JIT compile
	fmt.Println("Starting JIT compiler...")
	for _, c := range constants {
		if fn, ok := c.(*object.CompiledFunction); ok {
			mustJitCompile(fn)
		}
	}
	done <- struct{}{}
}

// TODO: add defer recover with logging to harden this
func mustJitCompile(fn *object.CompiledFunction) {
	machineCodeBuf := compileInstructions(fn)
	if len(machineCodeBuf) == 0 {
		return
	}

	execBuf := mustAllocateByMmap(len(machineCodeBuf))

	copy(execBuf, machineCodeBuf)

	mustMarkAsExecutable(execBuf)

	go func() {
		defer fn.JitInstructions.Unlock()
		fn.JitInstructions.Lock()
		fn.JitInstructions.MachineCodeInstructions = execBuf
	}()
}

func mustAllocateByMmap(size int) []byte {
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

// TODO: add logging and defer recover
func ExecMem(cl *object.Closure, sp *object.Object) {
	execMemInner(
		cl.Fn.JitInstructions.MachineCodeInstructions,
		uintptr(unsafe.Pointer(sp)),
		cl.Free,
	)
}

func execMemInner(mem []byte, sp uintptr, free []object.Object)

func compileInstructions(fn *object.CompiledFunction) []byte {
	// TODO: determine how to handle OpGetFree
	// idea - template that gets filled in by ExecMem
	if fn.Instructions.String() == "0000 OpNull\n0001 OpReturnValue\n" {
		// change vm[sp] to vm.NULL
		nullInterface := object.Object(object.NullS)
		nullP := uintptr(unsafe.Pointer(&nullInterface))
		movInsPtr := movWideVal(nullP, 3) // mov x3, #type(vm.NULL)
		movInsData := movWideVal(nullP + 8, 4) // mov x4, #vm.NULL
		storeInstructions := []byte {
			// TODO: fix the memory corruption this is causing vm.sp
			// 0x23, 0x00, 0x00, 0xf9, // str x3, [x1]
			// 0x24, 0x04, 0x00, 0xf9, // str x4, [x1, #8]
		}

		machineInstructions := append(movInsPtr, movInsData...)
		machineInstructions = append(machineInstructions, storeInstructions...)
		machineInstructions = append(machineInstructions, []byte{0x60, 0x02, 0x1f, 0xd6}...) // br x19
		return machineInstructions
	}
	return nil
}

// movWideVal puts a 64-bit value into general-purpose register #n
func movWideVal(val uintptr, register int) []byte {
	insBuf := []byte{}
	movz := binary.LittleEndian.Uint32([]byte{0x00, 0x00, 0x80, 0xd2})
	movk := binary.LittleEndian.Uint32([]byte{0x00, 0x00, 0x80, 0xf2})
	instructions := []uint32{movz, movk, movk, movk}
	mask := uintptr(256 * 256 - 1)
	for i, movIns := range instructions {
		movIns += uint32((val & mask) << 5)
		movIns += uint32(register)
		movIns += uint32(2097152 * i) // LSL #16/32/48
		insBuf = append(insBuf, convertInstructionToBytes(movIns)...)
		val >>= 16
	}
	return insBuf
}

func convertInstructionToBytes(instruction uint32) []byte {
	insBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(insBuf, instruction)
	return insBuf
}
