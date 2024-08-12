package vm

import (
	"fmt"
	"strings"

	"monkey/code"
	"monkey/compiler"
	"monkey/object"
)

const (
	GlobalsSize = 65536
	MaxFrames   = 1024
	StackSize   = 2048
)

var (
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
	NULL  = &object.Null{}
)

type VM struct {
	constants []object.Object
	globals   []object.Object

	stack []object.Object
	sp    int

	frames     []*Frame
	frameIndex int
}

func New(bytecode *compiler.Bytecode) *VM {
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
	}
}

func NewWithGlobalsStore(bytecode *compiler.Bytecode, s []object.Object) *VM {
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
	}
}

func (vm *VM) Run() error {
	var ip int
	var ins code.Instructions
	var op code.Opcode

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
				free[i] = vm.stack[vm.sp - numFree + i]
			}
			vm.sp = vm.sp - numFree
			if err := vm.push(&object.Closure{Fn: fn, Free: free}); err != nil {
				return err
			}
		case code.OpTrue:
			if err := vm.push(TRUE); err != nil {
				return err
			}
		case code.OpFalse:
			if err := vm.push(FALSE); err != nil {
				return err
			}
		case code.OpNull:
			if err := vm.push(NULL); err != nil {
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
				err = vm.push(FALSE)
			} else {
				err = vm.push(TRUE)
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
			scopeCount := code.ReadUint8(ins[ip+2:])
			vm.currentFrame().ip += 2

			vm.stack[vm.frames[vm.frameIndex-1-int(scopeCount)].basePointer + int(localIndex)] = vm.pop()
		case code.OpGetLocal:
			localIndex := code.ReadUint8(ins[ip+1:])
			vm.currentFrame().ip++

			localObj := vm.stack[vm.currentFrame().basePointer + int(localIndex)]
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
			err := vm.push(object.Array(arr))
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
			err := vm.push(object.Hash(hash))
			if err != nil {
				return err
			}
		case code.OpIndex:
			idxObj := vm.pop()
			containerObj := vm.pop()
			switch container := containerObj.(type) {
			case object.Array:
				intIdx, ok := idxObj.(*object.Integer)
				if !ok {
					return fmt.Errorf("cannot use an instance of type %T (%+v) as an array index",
						idxObj, idxObj)
				}
				arr := []object.Object(container)
				if 0 <= intIdx.Value && int(intIdx.Value) < len(arr) {
					if err := vm.push(arr[intIdx.Value]); err != nil {
						return err
					}
				} else {
					vm.push(NULL)
					return fmt.Errorf("index %d is out of bounds for an array with length %d",
						intIdx.Value, len(arr))
				}
			case object.Hash:
				hashableIdx, ok := idxObj.(object.Hashable)
				if !ok {
					return fmt.Errorf("cannot use an instance of type %T (%+v) as a hash index",
						idxObj, idxObj)
				}
				hash := map[object.HashKey]object.Object(container)
				val, ok := hash[hashableIdx.Hash()]
				if !ok {
					vm.push(NULL)
					return fmt.Errorf("index error for index %q", idxObj.Inspect())
				}
				if err := vm.push(val); err != nil {
					return err
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

			if err := vm.push(NULL); err != nil {
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
			return vm.push(TRUE)
		} else {
			return vm.push(FALSE)
		}
	case code.OpNeq:
		if leftVal != rightVal {
			return vm.push(TRUE)
		} else {
			return vm.push(FALSE)
		}
	case code.OpLessThan:
		if leftVal < rightVal {
			return vm.push(TRUE)
		} else {
			return vm.push(FALSE)
		}
	case code.OpLessThanEq:
		if leftVal <= rightVal {
			return vm.push(TRUE)
		} else {
			return vm.push(FALSE)
		}
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}

}

func (vm *VM) executeBooleanBinaryOp(lhs, rhs object.Object, op code.Opcode) error {
	switch op {
	case code.OpEq:
		if lhs == rhs {
			return vm.push(TRUE)
		} else {
			return vm.push(FALSE)
		}
	case code.OpNeq:
		if lhs != rhs {
			return vm.push(TRUE)
		} else {
			return vm.push(FALSE)
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
			return vm.push(TRUE)
		} else {
			return vm.push(FALSE)
		}
	case code.OpNeq:
		if leftVal != rightVal {
			return vm.push(TRUE)
		} else {
			return vm.push(FALSE)
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
	callee := vm.stack[vm.sp - numArgs - 1]
	switch callee := callee.(type) {
	case *object.Closure:
		if callee.Fn.NumParameters != numArgs {
			return fmt.Errorf("wrong number of arguments: want=%d, got=%d",
				callee.Fn.NumParameters, numArgs)
		}
		vm.pushFrame(NewFrame(callee, vm.sp - numArgs))
		vm.sp = vm.frames[vm.frameIndex - 1].basePointer + callee.Fn.NumLocals
		return nil
	case object.Builtin:
		arg := vm.stack[vm.sp - numArgs]
		argArray, ok := arg.(object.Array)
		if !ok {
			return fmt.Errorf("cannot call builtin with type %T, only object.Array", arg)
		}
		result := callee([]object.Object(argArray))
		vm.sp = vm.sp - numArgs - 1
		if result != nil {
			vm.push(result)
		} else {
			vm.push(NULL)
		}
		return nil
	default:
		return fmt.Errorf("calling non-function")
	}
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Integer:
		return obj.Value != 0
	case *object.Boolean:
		return obj != FALSE
	case *object.Null:
		return false
	default:
		panic(fmt.Sprintf("unsupported type cast to boolean: %T (%+v)", obj, obj))
	}
}
