package object

import (
	"testing"

	"monkey/code"
	"monkey/serializer"
)

func TestIntegerSerialization(t *testing.T) {
	values := []int64{
		0,
		1,
		-1,
		380580,
		-28508019,
	}

	for _, value := range values {
		a := &Integer{value}
		b := &Integer{}

		bs := a.Serialize()
		n := b.Deserialize(bs)
		if n != 9 {
			t.Errorf("only deserialized the first %d bytes of the buffer: %v", n, bs)
		}

		if !testObjectEquality(t, a, b) {
			t.Errorf("serialization of integers is incorrect: a=%+v, b=%+v", a, b)
		}
	}
}

func TestStringSerialization(t *testing.T) {
	strings := []string{
		"",
		"hello world",
		"hello, there!",
		"Hello, 世界",
	}

	for _, s := range strings {
		a := &String{s}
		b := &String{}

		bs := a.Serialize()
		b.Deserialize(bs)

		if !testObjectEquality(t, a, b) {
			t.Errorf("serialization of strings is incorrect: a=%+v, b=%+v", a, b)
		}
	}
}

func TestCompiledFunctionSerialization(t *testing.T) {
	functions := []CompiledFunction{
		{ // fn() {}
			Instructions: concatenateInstructions(
				code.Make(code.OpClosure, 0, 0),
				code.Make(code.OpPop),
			),
		},
		{ // fn(a, b) { let c = a + b; c }
			Instructions: concatenateInstructions(
				code.Make(code.OpClosure, 2, 0),
				code.Make(code.OpPop),
			),
			NumLocals: 1,
			NumParameters: 2,
		},
	}

	for _, f := range functions {
		a := &f
		b := &CompiledFunction{}

		bs := a.Serialize()
		b.Deserialize(bs)

		if !testObjectEquality(t, a, b) {
			t.Errorf("serialization of functions is incorrect: a=%+v, b=%+v", a, b)
		}
	}
}

func TestSerializationCorrectness(t *testing.T) {
	tests := []struct{
		object Object
		bs     []byte
	}{
		{
			object: &Integer{63},
			bs: []byte{0x01, 0x7e, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			object: &String{"hello world"},
			bs: []byte{
				0x02,
				0x16, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64,
			},
		},
		{
			object: &CompiledFunction{
				Instructions: concatenateInstructions(
					code.Make(code.OpTrue),
					code.Make(code.OpFalse),
					code.Make(code.OpEq),
					code.Make(code.OpReturnValue),
				),
				NumLocals: 0,
				NumParameters: 0,
			},
			bs: []byte{
				0x03,
				0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				byte(code.OpTrue), byte(code.OpFalse), byte(code.OpEq), byte(code.OpReturnValue),
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
	}

	for _, test := range tests {
		testObjectSerialization(t, test.object, test.bs)
	}
}

func testObjectSerialization(t *testing.T, obj Object, bs []byte) {
	var testBytes []byte
	switch obj := obj.(type) {
	case *Integer:
		if len(bs) != 9 {
			t.Fatalf("incorrect size, Integer byte size must equal 9. got=%d",
				len(bs))
		}
		if bs[0] != byte(serializer.INTEGER) {
			t.Fatalf("incorrect identifying integer byte, got=%d", bs[0])
		}
		testBytes = obj.Serialize()
	case *String:
		if bs[0] != byte(serializer.STRING) {
			t.Fatalf("incorrect identifying string byte, got=%d", bs[0])
		}
		testBytes = obj.Serialize()
	case *CompiledFunction:
		if bs[0] != byte(serializer.COMPILEDFN) {
			t.Fatalf("incorrect identifying function byte, got=%d", bs[0])
		}
		testBytes = obj.Serialize()
	}
	if len(bs) != len(testBytes) {
		t.Fatalf("incorrect length. expected=%d, got=%d", len(testBytes), len(bs))
	}
	for i := range bs {
		if bs[i] != testBytes[i] {
			t.Errorf("byte mismatch at position %d: expected=%d, got=%d",
				i, testBytes[i], bs[i])
		}
	}
}

func testObjectEquality(t *testing.T, a, b Object) bool {
	if a.Type() != b.Type() {
		t.Errorf("type mismatch: a=%s, b=%s", a.Type(), b.Type())
		return false
	}
	switch a := a.(type) {
	case *Integer:
		bVal := b.(*Integer).Value 
		if a.Value != bVal {
			t.Errorf("unequal integer values: a=%d, b=%d", a.Value, bVal)
			return false
		}
	case *String:
		bVal := b.(*String).Value
		if a.Value != bVal {
			t.Errorf("unequal string values: a=%q, b=%q", a.Value, bVal)
			return false
		}
	case *CompiledFunction:
		b := b.(*CompiledFunction)
		if len(a.Instructions) != len(b.Instructions) {
			t.Errorf("unequal instruction lengths: a=%d, b=%d",
				len(a.Instructions), len(b.Instructions))
			return false
		}
		for i := range a.Instructions {
			if a.Instructions[i] != b.Instructions[i] {
				t.Errorf("unequal instructions at position %d: a=%d, b=%d",
					i, a.Instructions[i], b.Instructions[i])
				return false
			}
		}
		if a.NumLocals != b.NumLocals {
			t.Errorf("unequal NumLocals: a=%d, b=%d",
				a.NumLocals, b.NumLocals)
			return false
		}
		if a.NumParameters != b.NumParameters {
			t.Errorf("unequal NumParameters: a=%d, b=%d",
				a.NumParameters, b.NumParameters)
			return false
		}
	default:
		t.Errorf("unhandled object type: %T", a)
		return false
	}
	return true
}

func concatenateInstructions(ins ...code.Instructions) code.Instructions {
	var bs []byte
	for _, instruction := range ins {
		bs = append(bs, []byte(instruction)...)
	}
	return code.Instructions(bs)
}
