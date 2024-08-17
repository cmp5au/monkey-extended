package vm

import (
	"fmt"
	"strings"

	"github.com/cmp5au/monkey-extended/code"
	"github.com/cmp5au/monkey-extended/compiler"
	"github.com/cmp5au/monkey-extended/experimental/jit"
	"github.com/cmp5au/monkey-extended/object"
)

const (
	GlobalsSize = 65536
	MaxFrames   = 1024
	StackSize   = 2048
)

type VM struct {
	constants []object.Object
	globals   []object.Object

	stack []object.Object
	sp    int

	frames     []*Frame
	frameIndex int

	// experimental; default off
	jitEnabled bool
}

func New(bytecode *compiler.Bytecode, jitEnabled bool) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	mainClosure := &object.Closure{Fn: mainFn}
	mainFrame := NewFrame(mainClosure, 0)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	return &VM{
		constants:  bytecode.Constants,
		globals:    make([]object.Object, GlobalsSize),
		stack:      make([]object.Object, StackSize),
		sp:         0,
		frames:     frames,
		frameIndex: 1,
		jitEnabled: jitEnabled,
	}
}

func NewWithGlobalsStore(bytecode *compiler.Bytecode, s []object.Object, jitEnabled bool) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	mainClosure := &object.Closure{Fn: mainFn}
	mainFrame := NewFrame(mainClosure, 0)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	return &VM{
		constants:  bytecode.Constants,
		globals:    s,
		stack:      make([]object.Object, StackSize),
		sp:         0,
		frames:     frames,
		frameIndex: 1,
		jitEnabled: jitEnabled,
	}
}

func (vm *VM) Run() error {
	var ip int
	var ins code.Instructions
	var op code.Opcode

	if vm.jitEnabled {
		jit.JitCompileFunctions(vm.constants)
	}

	for vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++

		ip = vm.currentFrame().ip
		ins = vm.currentFrame().Instructions()
		op = code.Opcode(ins[ip])

		switch op {
		case code.OpConstant:
			constIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2
			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}
		case code.OpClosure:
			constIndex := code.ReadUint16(ins[ip+1:])
			numFree := int(code.ReadUint8(ins[ip+3:]))
			vm.currentFrame().ip += 3
			fn, ok := vm.constants[constIndex].(*object.CompiledFunction)
			if !ok {
				return fmt.Errorf("cannot create a closure over anything except CompiledFunction, got=%T",
					vm.constants[constIndex])
			}
			free := make([]object.Object, numFree)
			for i := 0; i < numFree; i++ {
				free[i] = vm.stack[vm.sp-numFree+i]
			}
			vm.sp = vm.sp - numFree
			if err := vm.push(&object.Closure{Fn: fn, Free: free}); err != nil {
				return err
			}
		case code.OpTrue:
			if err := vm.push(object.TrueS); err != nil {
				return err
			}
		case code.OpFalse:
			if err := vm.push(object.FalseS); err != nil {
				return err
			}
		case code.OpNull:
			if err := vm.push(object.NullS); err != nil {
				return err
			}
		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpEq, code.OpNeq, code.OpLessThan, code.OpLessThanEq:
			if err := vm.executeInfixBinaryOp(op); err != nil {
				return err
			}
		case code.OpBang:
			obj := vm.pop()
			var err error
			if isTruthy(obj) {
				err = vm.push(object.FalseS)
			} else {
				err = vm.push(object.TrueS)
			}
			if err != nil {
				return err
			}
		case code.OpMinus:
			obj := vm.pop()
			intObj, ok := obj.(*object.Integer)
			if !ok {
				return fmt.Errorf("type mismatch, cannot prefix %T (%+v) with -",
					obj, obj)
			}
			if err := vm.push(&object.Integer{Value: -1 * intObj.Value}); err != nil {
				return err
			}
		case code.OpPop:
			vm.pop()
		case code.OpJump:
			vm.currentFrame().ip = int(code.ReadUint16(ins[ip+1:])) - 1
		case code.OpJumpNotTruthy:
			jumpIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2
			obj := vm.pop()
			if !isTruthy(obj) {
				vm.currentFrame().ip = int(jumpIndex) - 1
			}
		case code.OpSetGlobal:
			globalIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			vm.globals[globalIndex] = vm.pop()
		case code.OpGetGlobal:
			globalIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			err := vm.push(vm.globals[globalIndex])
			if err != nil {
				return err
			}
		case code.OpSetLocal:
			localIndex := code.ReadUint8(ins[ip+1:])
			vm.currentFrame().ip++

			vm.stack[vm.currentFrame().basePointer+int(localIndex)] = vm.pop()
		case code.OpGetLocal:
			localIndex := code.ReadUint8(ins[ip+1:])
			vm.currentFrame().ip++

			localObj := vm.stack[vm.currentFrame().basePointer+int(localIndex)]
			if err := vm.push(localObj); err != nil {
				return err
			}
		case code.OpGetBuiltin:
			builtinIndex := code.ReadUint8(ins[ip+1:])
			vm.currentFrame().ip++

			definition := object.Builtins[builtinIndex]

			if err := vm.push(definition.Builtin); err != nil {
				return err
			}
		case code.OpGetFree:
			freeIndex := code.ReadUint8(ins[ip+1:])
			vm.currentFrame().ip++
			freeVar := vm.currentFrame().cl.Free[freeIndex]
			if err := vm.push(freeVar); err != nil {
				return err
			}
		case code.OpCurrentClosure:
			if err := vm.push(vm.currentFrame().cl); err != nil {
				return err
			}
		case code.OpArray:
			length := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2
			arr := make([]object.Object, length, length)

			for i := range length {
				arr[length-1-i] = vm.pop()
			}
			arrObj := object.Array(arr)
			err := vm.push(&arrObj)
			if err != nil {
				return err
			}
		case code.OpHash:
			length := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2
			hash := map[object.HashKey]object.Object{}

			for _ = range length {
				value := vm.pop()
				key := vm.pop()
				hashableKey, ok := key.(object.Hashable)
				if !ok {
					return fmt.Errorf("cannot use an instance of type %T (%+v) as a hash key",
						key, key)
				}
				hash[hashableKey.Hash()] = value
			}
			hashObj := object.Hash(hash)
			err := vm.push(&hashObj)
			if err != nil {
				return err
			}
		case code.OpIndex:
			idxObj := vm.pop()
			containerObj := vm.pop()
			switch container := containerObj.(type) {
			case *object.Array:
				intIdx, ok := idxObj.(*object.Integer)
				if !ok {
					return fmt.Errorf("cannot use an instance of type %T (%+v) as an array index",
						idxObj, idxObj)
				}
				arr := []object.Object(*container)
				if 0 <= intIdx.Value && int(intIdx.Value) < len(arr) {
					if err := vm.push(arr[intIdx.Value]); err != nil {
						return err
					}
				} else if intIdx.Value < 0 && int(intIdx.Value) >= -1*len(arr) {
					if err := vm.push(arr[int(intIdx.Value)+len(arr)]); err != nil {
						return err
					}
				} else {
					vm.push(object.NullS)
					return fmt.Errorf("index %d is out of bounds for an array with length %d",
						intIdx.Value, len(arr))
				}
			case *object.Hash:
				hashableIdx, ok := idxObj.(object.Hashable)
				if !ok {
					return fmt.Errorf("cannot use an instance of type %T (%+v) as a hash index",
						idxObj, idxObj)
				}
				hash := map[object.HashKey]object.Object(*container)
				val, ok := hash[hashableIdx.Hash()]
				if !ok {
					vm.push(object.NullS)
					return fmt.Errorf("index error for index %q", idxObj.Inspect())
				}
				if err := vm.push(val); err != nil {
					return err
				}
			case *object.String:
				intIdx, ok := idxObj.(*object.Integer)
				if !ok {
					return fmt.Errorf("cannot use an instance of type %T (%+v) as a string index",
						idxObj, idxObj)
				}
				s := container.Value
				idx := int(intIdx.Value)
				if 0 <= idx && idx < len(s) {
					if err := vm.push(&object.String{string(s[idx])}); err != nil {
						return err
					}
				} else if idx < 0 && idx >= -1*len(s) {
					if err := vm.push(&object.String{string(s[idx+len(s)])}); err != nil {
						return err
					}
				} else {
					vm.push(object.NullS)
					return fmt.Errorf("index %d is out of bounds for an say with length %d",
						idx, len(s))
				}
			default:
				return fmt.Errorf("cannot index into an instance of type %T (%+v)",
					container, container)
			}
		case code.OpReturnValue:
			returnValue := vm.pop()

			vm.sp = vm.popFrame().basePointer - 1

			if err := vm.push(returnValue); err != nil {
				return err
			}
		case code.OpReturn:
			vm.sp = vm.popFrame().basePointer - 1

			if err := vm.push(object.NullS); err != nil {
				return err
			}
		case code.OpCall:
			numArgs := code.ReadUint8(ins[ip+1:])
			vm.currentFrame().ip++
			if err := vm.callFunction(int(numArgs)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (vm *VM) StackTop() object.Object {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) ViewStack() string {
	objects := []string{}
	for _, obj := range vm.stack[:vm.sp+4] {
		if obj != nil {
			objects = append(objects, fmt.Sprintf("%+v", obj))
		} else {
			objects = append(objects, "")
		}
	}
	return "[ " + strings.Join(objects, ", ") + " ]"
}

func (vm *VM) push(obj object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}
	vm.stack[vm.sp] = obj
	vm.sp++

	return nil
}

func (vm *VM) pop() object.Object {
	vm.sp--
	return vm.stack[vm.sp]
}

func (vm *VM) executeInfixBinaryOp(op code.Opcode) error {
	rhs := vm.pop()
	lhs := vm.pop()

	lhsType := lhs.Type()

	if lhsType != rhs.Type() {
		return fmt.Errorf("type mismatch: %s %s", lhsType, rhs.Type())
	}
	switch lhsType {
	case object.INTEGER:
		return vm.executeIntegerBinaryOp(lhs, rhs, op)
	case object.BOOLEAN:
		return vm.executeBooleanBinaryOp(lhs, rhs, op)
	case object.STRING:
		return vm.executeStringBinaryOp(lhs, rhs, op)
	}
	return fmt.Errorf("unsupported types for binary operation: %T %d %T", lhs, op, rhs)
}

func (vm *VM) executeIntegerBinaryOp(lhs, rhs object.Object, op code.Opcode) error {
	leftVal := lhs.(*object.Integer).Value
	rightVal := rhs.(*object.Integer).Value

	switch op {
	case code.OpAdd:
		return vm.push(&object.Integer{Value: leftVal + rightVal})
	case code.OpSub:
		return vm.push(&object.Integer{Value: leftVal - rightVal})
	case code.OpMul:
		return vm.push(&object.Integer{Value: leftVal * rightVal})
	case code.OpDiv:
		return vm.push(&object.Integer{Value: leftVal / rightVal})
	case code.OpEq:
		if leftVal == rightVal {
			return vm.push(object.TrueS)
		} else {
			return vm.push(object.FalseS)
		}
	case code.OpNeq:
		if leftVal != rightVal {
			return vm.push(object.TrueS)
		} else {
			return vm.push(object.FalseS)
		}
	case code.OpLessThan:
		if leftVal < rightVal {
			return vm.push(object.TrueS)
		} else {
			return vm.push(object.FalseS)
		}
	case code.OpLessThanEq:
		if leftVal <= rightVal {
			return vm.push(object.TrueS)
		} else {
			return vm.push(object.FalseS)
		}
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}

}

func (vm *VM) executeBooleanBinaryOp(lhs, rhs object.Object, op code.Opcode) error {
	switch op {
	case code.OpEq:
		if lhs == rhs {
			return vm.push(object.TrueS)
		} else {
			return vm.push(object.FalseS)
		}
	case code.OpNeq:
		if lhs != rhs {
			return vm.push(object.TrueS)
		} else {
			return vm.push(object.FalseS)
		}
	default:
		return fmt.Errorf("unknown boolean operator: %d", op)
	}
}

func (vm *VM) executeStringBinaryOp(lhs, rhs object.Object, op code.Opcode) error {
	leftVal := lhs.(*object.String).Value
	rightVal := rhs.(*object.String).Value

	switch op {
	case code.OpEq:
		if leftVal == rightVal {
			return vm.push(object.TrueS)
		} else {
			return vm.push(object.FalseS)
		}
	case code.OpNeq:
		if leftVal != rightVal {
			return vm.push(object.TrueS)
		} else {
			return vm.push(object.FalseS)
		}
	case code.OpLessThan:
		if leftVal < rightVal {
			return vm.push(object.TrueS)
		} else {
			return vm.push(object.FalseS)
		}
	case code.OpLessThanEq:
		if leftVal <= rightVal {
			return vm.push(object.TrueS)
		} else {
			return vm.push(object.FalseS)
		}
	case code.OpAdd:
		return vm.push(&object.String{Value: leftVal + rightVal})
	default:
		return fmt.Errorf("unknown string operator: %d", op)
	}
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.frameIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.frameIndex] = f
	vm.frameIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.frameIndex--
	return vm.frames[vm.frameIndex]
}

func (vm *VM) callFunction(numArgs int) error {
	callee := vm.stack[vm.sp-numArgs-1]
	switch callee := callee.(type) {
	case *object.Closure:
		if callee.Fn.NumParameters != numArgs {
			return fmt.Errorf("wrong number of arguments: want=%d, got=%d",
				callee.Fn.NumParameters, numArgs)
		}
		if vm.jitEnabled && vm.callFunctionViaJit(callee) {
			return nil
		}
		vm.pushFrame(NewFrame(callee, vm.sp-numArgs))
		vm.sp = vm.frames[vm.frameIndex-1].basePointer + callee.Fn.NumLocals
		return nil
	case object.Builtin:
		arg := vm.stack[vm.sp-numArgs]
		argArray, ok := arg.(*object.Array)
		if !ok {
			return fmt.Errorf("cannot call builtin with type %T, only object.Array", arg)
		}
		result := callee([]object.Object(*argArray))
		vm.sp = vm.sp - numArgs - 1
		if result != nil {
			vm.push(result)
		} else {
			vm.push(object.NullS)
		}
		return nil
	default:
		return fmt.Errorf("calling non-function")
	}
}

func (vm *VM) callFunctionViaJit(callee *object.Closure) (success bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			success = false
		}
	}()
	defer callee.Fn.JitInstructions.Unlock()
	callee.Fn.JitInstructions.Lock()
	if callee.Fn.JitInstructions.MachineCodeInstructions == nil {
		return false
	}

	jit.ExecMem(callee, &vm.stack[vm.sp])
	vm.sp -= callee.Fn.NumParameters
	vm.stack[vm.sp] = vm.stack[vm.sp + callee.Fn.NumParameters]
	return true
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Integer:
		return obj.Value != 0
	case *object.Boolean:
		return obj != object.FalseS
	case *object.Null:
		return false
	default:
		panic(fmt.Sprintf("unsupported type cast to boolean: %T (%+v)", obj, obj))
	}
}
