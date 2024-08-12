package vm

import (
	"fmt"
	"testing"

	"monkey/ast"
	"monkey/compiler"
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
)

type vmTestCase struct {
	input    string
	expected interface{}
}

func TestIntegerArithmetic(t *testing.T) {
	tests := []vmTestCase{
		{"1", 1},
		{"2", 2},
		{"1 + 2", 3},
		{"1 * 2 + 6 / 3 - 4", 0},
		{"1 - -1", 2},
		{"-2 * -(0 - -6)", 12},
		{"[1 + 2, 3 * 4, 5 - 6][8 - 7]", 12},
	}

	runVmTests(t, tests)
}

func TestBooleanExpressions(t *testing.T) {
	tests := []vmTestCase{
		{"true;", true},
		{"false;", false},
		{"1 < 2", true},
		{"3 == 4", false},
		{"5 != -6", true},
		{"7 > 8", false},
		{"-9 <= 10", true},
		{"11 >= 12", false},
		{"true == true", true},
		{"false != false", false},
		{"!true", false},
		{"!false", true},
	}

	runVmTests(t, tests)
}

func TestConditionals(t *testing.T) {
	tests := []vmTestCase{
		{"if (true) { 10 }", 10},
		{"if (true) { 10 } else { 20 }", 10},
		{"if (false) { 10 } else { 20 }", 20},
		{"if (false) { 10 }", NULL},
		{"if (1 < 2) { 10 }", 10},
		{"if (1 < 2) { 10 } else { 20 }", 10},
		{"if (1 > 2) { 10 } else { 20 }", 20},
		{"if (1 > 2) { 10 }", NULL},
		{"if (1) { 10 }", 10},
		{"if (0) { 10 }", NULL},
		{"if (null) { 1 } else { null }", NULL},
	}

	runVmTests(t, tests)
}

func TestGlobalLetStatements(t *testing.T) {
	tests := []vmTestCase{
		{"let one = 1; one", 1},
		{"let one = 1; let two = 2; one + two", 3},
		{"let one = 1; let two = one + one; one + two", 3},
	}

	runVmTests(t, tests)
}

func TestAssignmentStatements(t *testing.T) {
	tests := []vmTestCase{
		{ // test setting global from within enclosed scope
			input: `
			let a = 10;
			let f = fn() {
				let b = 100;
				let g = fn() {
					a = a + 1;
					let c = 10000;
					return c;
				};
				return g() + a * b;
			};
			f();
			`,
			expected: 11100, // 10000 + 11 * 100 = 11100
		},
		{ // test setting local from within enclosed scope
			input: `
			let a = 10;
			let f = fn() {
				let b = 100;
				let g = fn() {
					b = b + 1;
					let c = 10000;
					return c;
				};
				return g() + a * b;
			};
			f();
			`,
			expected: 11000, // 10000 + 10 * 100 = 11000
		},
	}

	runVmTests(t, tests)
}


func TestStringExpression(t *testing.T) {
	tests := []vmTestCase{
		{`"monkey"`, "monkey"},
		{`"mon" + "key"`, "monkey"},
		{`{"monkey": "lang"}["monkey"]`, "lang"},
	}

	runVmTests(t, tests)
}

func TestArrayLiterals(t *testing.T) {
	tests := []vmTestCase{
		{"[]", []int{}},
		{"[1, 2, 3]", []int{1, 2, 3}},
		{"[1 + 2, 3 * 4, 5 - 6]", []int{3, 12, -1}},
	}

	runVmTests(t, tests)
}

func TestHashLiterals(t *testing.T) {
	tests := []vmTestCase{
		{
			"{}", map[string]object.Object{},
		},
		{
			`{"a": 1, "b": 2}`,
			map[string]object.Object{
				"a": &object.Integer{Value: 1},
				"b": &object.Integer{Value: 2},
			},
		},
		{
			`{"a" + "a": 2 * 2}`,
			map[string]object.Object{
				"aa": &object.Integer{Value: 4},
			},
		},
		{`{1: {"c": 3}}[1]`, map[string]object.Object{"c": &object.Integer{3}}},
		{`{true: [1, 2, 3]}[true]`, []int{1, 2, 3}},
	}

	runVmTests(t, tests)
}

func TestFunctions(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let fivePlusTen = fn() { 5 + 10; };
			fivePlusTen();
			`,
			expected: 15,
		},
		{
			input: `
			let one = fn() { let one = 1; one };
			one();
			`,
			expected: 1,
		},
		{
			input: `
			let oneAndTwo = fn() { let one = 1; let two = 2; one + two; };
			oneAndTwo();
			`,
			expected: 3,
		},
		{
			input: `
			let oneAndTwo = fn() { let one = 1; let two = 2; one + two; };
			let threeAndFour = fn() { let three = 3; let four = 4; three + four; };
			threeAndFour() - oneAndTwo();
			`,
			expected: 4,
		},
		{
			input: `
			let firstFoobar = fn() { let foobar = 50; foobar; };
			let secondFoobar = fn() { let foobar = 100; foobar; };
			firstFoobar() + secondFoobar();
			`,
			expected: 150,
		},
		{
			input: `
			let globalSeed = 50;
			let minusOne = fn() {
				let num = 1;
				globalSeed - num;
			}
			let minusTwo = fn() {
				let num = 2;
				globalSeed - num;
			}
			minusOne() + minusTwo();
			`,
			expected: 97,
		},
		{
			input: `
			let identity = fn(a) { a; };
			identity(5);
			`,
			expected: 5,
		},
		{
			input: `
			let add = fn(a, b) { a + b; };
			add(34, 35);
			`,
			expected: 69,
		},
		{
			input: `
			let globalNum = 10;
			let sum = fn(a, b) {
				let c = a + b;
				c + globalNum;
			};

			let outer = fn() {
				sum(1, 2) + sum(3, 4) + globalNum;
			};

			outer() + globalNum;
			`,
			expected: 50,
		},
		{
			input: `fn(){}()`,
			expected: NULL,
		},
	}

	runVmTests(t, tests)
}

func TestCallingFunctionsWithWrongArguments(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `fn() { 1; }(1);`,
			expected: `wrong number of arguments: want=0, got=1`,
		},
		{
			input: `fn(a) { a; }();`,
			expected: `wrong number of arguments: want=1, got=0`,
		},
		{
			input: `fn(a, b) { a + b; }(1);`,
			expected: `wrong number of arguments: want=2, got=1`,
		},
	}

	for _, test := range tests {
		program := parse(test.input)
		comp := compiler.New()
		if err := comp.Compile(program); err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		vm := New(comp.Bytecode())
		err := vm.Run()
		if err == nil {
			t.Fatalf("expected VM error but resulted in none.")
		}
		if err.Error() != test.expected {
			t.Fatalf("wrong VM error: want=%q, got=%q", test.expected, err)
		}
	}
}

func TestBuiltinFunctions(t *testing.T) {
	tests := []vmTestCase{
		{`len("")`, 0},
		{`len("four")`, 4},
		{`len("Hello, world!")`, 13},
		{
			`len(1)`,
			&object.Error{
				Message: "len() argument must be iterable",
			},
		},
		{
			`len("one", "two")`,
			&object.Error{
				Message: "len() takes 1 argument",
			},
		},
		{`len([1, 2, 3])`, 3},
		{`len([])`, 0},
		{`puts("hello, world!")`, NULL},
		{`push([], 1)`, []int{1}},
		{`push(1, 1)`, &object.Error{"first argument to push() must be an array"}},
	}

	runVmTests(t, tests)
}

func TestClosures(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let newClosure = fn(a) {
				fn() { a; };
			};
			let closure = newClosure(99);
			closure();
			`,
			expected: 99,
		},
		{
			input: `
			let newAdder = fn(a, b) {
				fn(c) { a + b + c };
			};
			let adder = newAdder(1, 2);
			adder(8);
			`,
			expected: 11,
		},
		{
			input: `
			let newAdder = fn(a, b) {
				let c = a + b;
				fn(d) { c + d };
			};
			let adder = newAdder(30, 31);
			adder(8);
			`,
			expected: 69,
		},
		{
			input: `
			let newAdderOuter = fn(a, b) {
				let c = a + b;
				fn(d) {
					let e = d + c;
					fn(f) { e + f; };
				};
			};
			let newAdderInner = newAdderOuter(1, 2);
			let adder = newAdderInner(3);
			adder(8);
			`,
			expected: 14,
		},
		{
			input: `
			let a = 1;
			let newAdderOuter = fn(b) {
				fn(c) {
					fn(d) { a + b + c + d };
				};
			};
			let newAdderInner = newAdderOuter(2);
			let adder = newAdderInner(3);
			adder(8);
			`,
			expected: 14,
		},
		{
			input: `
			let newClosure = fn(a, b) {
				let one = fn() { a; };
				let two = fn() { b; };
				fn() { one() + two(); };
			};
			let closure = newClosure(9, 90);
			closure();
			`,
			expected: 99,
		},
	}

	runVmTests(t, tests)
}

func TestRecursiveFunctions(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let countDown = fn(x) {
				if (x == 0) {
					return 0;
				} else {
					countDown(x - 1);
				}
			};
			countDown(1);
			`,
			expected: 0,
		},
		{
			input: `
			let countDown = fn(x) {
				if (x == 0) {
					return 0;
				} else {
					countDown(x - 1);
				}
			};
			let wrapper = fn() {
				countDown(1);
			};
			wrapper();
			`,
			expected: 0,
		},
		{
			input: `
			let wrapper = fn() {
				let countDown = fn(x) {
					if ( x == 0) {
						return 0;
					} else {
						countDown(x - 1);
					}
				};
				countDown(1);
			};
			wrapper();
			`,
			expected: 0,
		},
	}

	runVmTests(t, tests)
}

func TestRecursiveFibonacci(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let fibonacci = fn(x) {
				if (x == 0) {
					return 0;
				} else {
					if (x == 1) {
						return 1;
					} else {
						fibonacci(x - 1) + fibonacci(x - 2);
					}
				}
			};
			fibonacci(15);
			`,
			expected: 610,
		},
	}

	runVmTests(t, tests)
}

func TestForStatements(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let x = 0;
			for (x < 5) {
				let x = x + 1;
			};
			x;
			`,
			expected: 5,
		},
		{ // collatz for loop
			input: `
			let count = 1;
			let x = 9;
			for (x != 1) {
				if (x == x / 2 * 2) {
					let x = x / 2;
				} else {
					let x = 3 * x + 1;
				};
				let count = count + 1;
			};
			count;
			`,
			expected: 20,
		},
		{ // bitmasking to test break and continue
			input: `
			let x = 0;
			let i = 0;
			for {
				let i = i + 1;
				let x = 2 * x;
				if (i == i / 2 * 2) {
					continue;
				};
				let x = x + 1;
				if (x > 16) {
					break;
				};
			};
			x;
			`,
			expected: 21, // 10101 binary = 21 decimal
		},
	}

	runVmTests(t, tests)
}

func runVmTests(t *testing.T, tests []vmTestCase) {
	t.Helper()

	for _, test := range tests {
		program := parse(test.input)
		c := compiler.New()
		err := c.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		vm := New(c.Bytecode())
		if err = vm.Run(); err != nil {
			t.Fatalf("vm error: %s", err)
		}

		stackElem := vm.LastPoppedStackElem()

		testExpectedObject(t, test.expected, stackElem)
	}
}

func testExpectedObject(t *testing.T, expected interface{}, actual object.Object) {
	t.Helper()

	switch expected := expected.(type) {
	case int:
		err := testIntegerObject(int64(expected), actual)
		if err != nil {
			t.Errorf("testIntegerObject failed: %s", err)
		}
	case bool:
		err := testBooleanObject(expected, actual)
		if err != nil {
			t.Errorf("testBooleanObject failed: %s", err)
		}
	case string:
		err := testStringObject(expected, actual)
		if err != nil {
			t.Errorf("testStringObject failed: %s", err)
		}
	case []int:
		err := testArrayObject(expected, actual)
		if err != nil {
			t.Errorf("testArrayObject failed: %s", err)
		}
	case map[string]object.Object:
		err := testHashObject(expected, actual)
		if err != nil {
			t.Errorf("testHashObject failed: %s", err)
		}
	case *object.Null:
		if actual != NULL {
			t.Errorf("object is not Null: %T (%+v)", actual, actual)
		}
	case *object.Error:
		errObj, ok := actual.(*object.Error)
		if !ok {
			t.Errorf("object is not Error: %T (%+v)", actual, actual)
			return
		}
		if errObj.Message != expected.Message {
			t.Errorf("wrong error message. expected=%q, got=%q",
				expected.Message, errObj.Message)
		}
	default:
		t.Errorf("unexpected type: %T (%+v)", expected, expected)
	}
}

func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

// tests an expected int64 constant against the actual Object constant
// in the compiled bytecode
func testIntegerObject(expected int64, actual object.Object) error {
	result, ok := actual.(*object.Integer)
	if !ok {
		return fmt.Errorf("object is not Integer.\ngot=%T (%+v)",
			actual, actual)
	}
	if result.Value != expected {
		return fmt.Errorf("object has wrong value.\nexpected=%d\ngot=%d",
			expected, result.Value)
	}
	return nil
}

// tests an expected bool constant against the actual Object constant
// in the compiled bytecode
func testBooleanObject(expected bool, actual object.Object) error {
	result, ok := actual.(*object.Boolean)
	if !ok {
		return fmt.Errorf("object is not Boolean.\ngot=%T (%+v)",
			actual, actual)
	}
	if result.Value != expected {
		return fmt.Errorf("object has wrong value.\nexpected=%v\ngot=%v",
			expected, result.Value)
	}
	return nil
}

// tests an expected string constant against the actual Object constant
// in the compiled bytecode
func testStringObject(expected string, actual object.Object) error {
	result, ok := actual.(*object.String)
	if !ok {
		return fmt.Errorf("object is not String.\ngot=%T (%+v)",
			actual, actual)
	}
	if result.Value != expected {
		return fmt.Errorf("object has wrong value.\nexpected=%q\ngot=%q",
			expected, result.Value)
	}
	return nil
}

// tests an expected array against the actual Object in the compiled bytecode
func testArrayObject(expected []int, actual object.Object) error {
	result, ok := actual.(object.Array)
	if !ok {
		return fmt.Errorf("object is not Array.\ngot=%T (%+v)",
			actual, actual)
	}
	arr := []object.Object(result)
	if len(expected) != len(arr) {
		return fmt.Errorf("wrong number of elements: expected=%d, got=%d",
			len(expected), len(arr))
	}
	for i := range expected {
		if err := testIntegerObject(int64(expected[i]), arr[i]); err != nil {
			return err
		}
	}
	return nil
}

// tests an expected hashmap against the actual Object in the compiled bytecode
func testHashObject(expected map[string]object.Object, actual object.Object) error {
	result, ok := actual.(object.Hash)
	if !ok {
		return fmt.Errorf("object is not Hash.\ngot=%T (%+v)",
			actual, actual)
	}
	hash := map[object.HashKey]object.Object(result)
	if len(expected) != len(hash) {
		return fmt.Errorf("wrong number of elements: expected=%d, got=%d",
			len(expected), len(hash))
	}
	for key, val := range expected {
		sKey := &object.String{key}
		actualObj, ok := hash[sKey.Hash()]
		if !ok {
			return fmt.Errorf("expected key=%q not present in object.Hash", key)
		}
		expectedVal := val.(*object.Integer).Value
		actualVal := actualObj.(*object.Integer).Value
		if expectedVal != actualVal {
			return fmt.Errorf("hash value mismatch for key=%q: expected=%d, got=%d", key, expectedVal, actualVal)
		}
	}

	return nil
}
