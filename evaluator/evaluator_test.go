package evaluator

import (
	"strings"
	"testing"

	"github.com/cmp5au/monkey-extended/lexer"
	"github.com/cmp5au/monkey-extended/object"
	"github.com/cmp5au/monkey-extended/parser"
)

type evaluatorTest struct {
	input    string
	expected interface{}
}

type expectedFn struct {
	parameters []string
	body       string
}

func TestEvalIntegerExpression(t *testing.T) {
	tests := []evaluatorTest{
		{"5", 5},
		{"10", 10},
		{"-5", -5},
		{"-10", -10},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"-50 + 100 - 50", 0},
		{"5 * 2 + 10", 20},
		{"5 + 2 * 10", 25},
		{"20 + 2 * -10", 0},
		{"50 / 2 * 2 + 10", 60},
		{"2 * (5 + 10)", 30},
		{"3 * 3 * 3 + 10", 37},
		{"3 * (3 * 3) + 10", 37},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
	}

	runEvaluatorTests(t, tests)
}

func TestEvalBooleanExpression(t *testing.T) {
	tests := []evaluatorTest{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 < 1", false},
		{"1 > 1", false},
		{"1 == 1", true},
		{"1 != 1", false},
		{"1 == 2", false},
		{"1 != 2", true},
		{"true == true", true},
		{"false == false", true},
		{"true == false", false},
		{"true != false", true},
		{"false != true", true},
		{"(1 < 2) == true", true},
		{"(1 < 2) == false", false},
		{"(1 > 2) == true", false},
		{"(1 > 2) == false", true},
		{`"a" < "b"`, true},
		{`"a" >= "b"`, false},
		{`"a" <= "a"`, true},
		{`"b" > "b"`, false},
	}

	runEvaluatorTests(t, tests)
}

func TestEvalStringExpression(t *testing.T) {
	tests := []evaluatorTest{
		{"\"hi\"", "hi"},
		{"\"a\" + \"b\"", "ab"},
		{"\"a\" < \"b\"", true},
		{"\"a\" > \"b\"", false},
		{"\"a\" <= \"b\"", true},
		{"\"a\" >= \"b\"", false},
		{"\"a\" == \"b\"", false},
		{"\"a\" != \"b\"", true},
		{`"hello"[3]`, "l"},
		{`let x = "hello, world!"; x[12];`, "!"},
	}

	runEvaluatorTests(t, tests)
}

func TestEvalArrayExpression(t *testing.T) {
	tests := []evaluatorTest{
		{"[1, 2, 3]", []int{1, 2, 3}},
		{"[1, 2, 3][1]", 2},
		{"[]", []int{}},
		{"push([1, 2, 3], 4)", []int{1, 2, 3, 4}},
	}

	runEvaluatorTests(t, tests)
}

func TestEvalHashExpression(t *testing.T) {
	tests := []evaluatorTest{
		{`{}`, map[string]object.Object{}},
		{
			input: `{"a": 1, "b": 2}`,
			expected: map[string]object.Object{
				"a": &object.Integer{1},
				"b": &object.Integer{2},
			},
		},
		{`{"a": 1, "b": 2}["b"]`, 2},
		{`{1: {"c": 3}}[1]`, map[string]object.Object{"c": &object.Integer{3}}},
		{`{true: [1, 2, 3]}[true]`, []int{1, 2, 3}},
	}

	runEvaluatorTests(t, tests)
}

func TestBangPrefixUnaryOp(t *testing.T) {
	tests := []evaluatorTest{
		{"!null", true},
		{"!true", false},
		{"!false", true},
		{"!5", false},
		{"!!true", true},
		{"!!false", false},
		{"!!5", true},
	}

	runEvaluatorTests(t, tests)
}

func TestIfElseExpressions(t *testing.T) {
	tests := []evaluatorTest{
		{"if (true) { 10 }", 10},
		{"if (false) { 10 }", nil},
		{"if (1) { 10 }", 10},
		{"if (1 < 2) { 10 }", 10},
		{"if (1 > 2) { 10 }", nil},
		{"if (1 > 2) { 10 } else { 20 }", 20},
		{"if (1 < 2) { 10 } else { 20 }", 10},
	}

	runEvaluatorTests(t, tests)
}

func TestForStatements(t *testing.T) {
	tests := []evaluatorTest{
		{"let i = 0; for (i < 1) { let i = i + 1 }; i;", 1},
		{"for (true) { break; };", nil},
		{"let i = 0; for (i < 2) { let i = i + 1; continue; break; }; i;", 2},
		{"let i = 0; for (i < 2) { let i = i + 1; break; continue; }; i;", 1},
	}

	runEvaluatorTests(t, tests)
}

func TestReturnStatements(t *testing.T) {
	tests := []evaluatorTest{
		{"return null;", nil},
		{"return 10;", 10},
		{"return 10; 9;", 10},
		{"return 2 * 5; 9;", 10},
		{"9; return 2 * 5; 9;", 10},
		{
			`
				if (10 > 1) {
					if (10 > 1) {
						return 10;
					}

					return 1;
				}
			`,
			10,
		},
	}

	runEvaluatorTests(t, tests)
}

func TestErrorHandling(t *testing.T) {
	tests := []evaluatorTest{
		{
			"5 + true;",
			&object.Error{"type mismatch: INTEGER + BOOLEAN"},
		},
		{
			"5 + true; 5;",
			&object.Error{"type mismatch: INTEGER + BOOLEAN"},
		},
		{
			"-true",
			&object.Error{"unknown operator: -BOOLEAN"},
		},
		{
			"true + false;",
			&object.Error{"unknown operator: BOOLEAN + BOOLEAN"},
		},
		{
			"5; true + false; 5",
			&object.Error{"unknown operator: BOOLEAN + BOOLEAN"},
		},
		{
			"if (10 > 1) { true + false; }",
			&object.Error{"unknown operator: BOOLEAN + BOOLEAN"},
		},
		{
			`
			if (10 > 1) {
				if (10 > 1) {
					return true + false;
				}

				return 1;
			}
			`,
			&object.Error{"unknown operator: BOOLEAN + BOOLEAN"},
		},
		{
			"foobar",
			&object.Error{"identifier not found: foobar"},
		},
		{
			"a = 0",
			&object.Error{"identifier a has not been declared in scope"},
		},
	}

	runEvaluatorTests(t, tests)
}

func TestLetStatements(t *testing.T) {
	tests := []evaluatorTest{
		{"let a;", nil},
		{"let a; a;", nil},
		{"let a = 5;", nil},
		{"let a = 5; a;", 5},
		{"let a = 5 * 5; a;", 25},
		{"let a = 5; let b = a; b;", 5},
		{"let a = 5; let b = a; let c = a + b + 5; c;", 15},
	}

	runEvaluatorTests(t, tests)
}

func TestAssignmentStatements(t *testing.T) {
	tests := []evaluatorTest{
		{"let a = 0; a = 5;", nil},
		{"let a = 0; a = 5; a;", 5},
		{"let a = 0; a = 5 * 5; a;", 25},
		{"let a = 0; a = 5; let b = 2; b = a; b;", 5},
		{"let a = 0; a = 5; let b = 2; b = a; let c = 4; c = a + b + 5; c;", 15},
	}

	runEvaluatorTests(t, tests)
}

func TestFunctionApplication(t *testing.T) {
	tests := []evaluatorTest{
		{"let identity = fn(x) { x; }; identity(5);", 5},
		{"let identity = fn(x) { return x; }; identity(5);", 5},
		{"let double = fn(x) { x * 2; }; double(5);", 10},
		{"let add = fn(x, y) { x + y; }; add(5, 5);", 10},
		{"let add = fn(x, y) { x + y; }; add(5 + 5, add(5, 5));", 20},
		{"fn(x) { x; }(5)", 5},
	}

	runEvaluatorTests(t, tests)
}

func TestEnclosingEnvironments(t *testing.T) {
	tests := []evaluatorTest{
		{
			input: `
let first = 10;
let second = 10;
let third = 10;

let ourFunction = fn(first) {
  let second = 20;

  first + second + third;
};

ourFunction(20) + first + second;
			`,
			expected: 70,
		},
	}

	runEvaluatorTests(t, tests)
}

func TestBuiltinFunctions(t *testing.T) {
	tests := []evaluatorTest{
		{"len(\"Hello, world!\");", 13},
		{"puts(\"Hello, world!\");", nil},
		{"let x = [1]; let y = push(x, 2); y[1];", 2},
		{"let arr = [1, 2, 3]; pop(arr); arr;", []int{1, 2}},
		{"let arr = [1, 2, 3]; del(arr, 1);", nil},
		{"let arr = [1, 2, 3]; del(arr, 1); arr;", []int{1, 3}},
		{`let hash = {"a": 1, true: 2}; del(hash, "a");`, nil},
		{
			input: `let hash = {"a": 1, true: 2}; del(hash, true); hash`,
			expected: map[string]object.Object{
				"a": &object.Integer{1},
			},
		},
		{
			input:    "let x = 1; del(x);",
			expected: &object.Error{"del() takes 2 arguments"},
		},
		{
			input: "let x = 1; del(x, 1);",
			expected: &object.Error{
				Message: "first argument to del() must be an Array or Hash",
			},
		},
		{
			input: `let hash = {"a": 1, true: 2}; del(hash, ["a"]);`,
			expected: &object.Error{
				Message: "cannot delete non-hashable key of type *object.Array from Hash",
			},
		},
		{
			input:    `let hash = {"a": 1, true: 2}; del(hash, "b");`,
			expected: &object.Error{`entry "b" not found in Hash`},
		},
		{
			input:    `let arr = [1, 2, 3]; del(arr, 4);`,
			expected: &object.Error{"index 4 is not valid for an Array of length 3"},
		},
		{`let arr = [1, 2, 3]; pushleft(arr, 0);`, []int{0, 1, 2, 3}},
		{`let arr = [1, 2, 3]; pushleft(arr, 0); arr;`, []int{0, 1, 2, 3}},
		{`let arr = [1, 2, 3]; popleft(arr);`, 1},
		{`let arr = [1, 2, 3]; popleft(arr); arr;`, []int{2, 3}},
	}

	runEvaluatorTests(t, tests)
}

func runEvaluatorTests(t *testing.T, tests []evaluatorTest) {
	for _, test := range tests {
		evaluated := testEvaluate(test.input)
		switch expected := test.expected.(type) {
		case string:
			testStringObject(t, evaluated, expected)
		case bool:
			testBooleanObject(t, evaluated, expected)
		case int:
			testIntegerObject(t, evaluated, int64(expected))
		case nil:
			testNullObject(t, evaluated)
		case []int:
			testArrayObject(t, evaluated, expected)
		case map[string]object.Object:
			testHashObject(t, evaluated, expected)
		case expectedFn:
			testFunctionObject(t, evaluated, expected)
		case *object.Error:
			testErrorObject(t, evaluated, expected)
		default:
			t.Fatalf("couldn't evaluate expected value of type %T (%+v)",
				expected, expected)
		}
	}
}

func testNullObject(t *testing.T, obj object.Object) bool {
	if obj != object.NullS {
		t.Errorf("object is not NULL. got=%T (%+v)", obj, obj)
		return false
	}
	return true
}

func testEvaluate(input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		panic(strings.Join(p.Errors(), "; "))
	}

	env := object.NewEnvironment()
	return Evaluate(program, env)
}

func testIntegerObject(t *testing.T, evaluated object.Object, expected int64) bool {
	intObj, ok := evaluated.(*object.Integer)
	if !ok {
		t.Errorf("object is not *object.Integer, got=%T (%+v)", evaluated, evaluated)
		return false
	}

	if intObj.Value != expected {
		t.Errorf("object has the wrong value, expected=%d, got=%d", expected, intObj.Value)
		return false
	}

	return true
}

func testBooleanObject(t *testing.T, evaluated object.Object, expected bool) bool {
	boolObj, ok := evaluated.(*object.Boolean)
	if !ok {
		t.Errorf("object is not *object.Boolean, got=%T (%+v)", evaluated, evaluated)
		return false
	}

	if boolObj.Value != expected {
		t.Errorf("object has the wrong value, expected=%t, got=%t", expected, boolObj.Value)
		return false
	}

	return true
}

func testStringObject(t *testing.T, evaluated object.Object, expected string) bool {
	stringObj, ok := evaluated.(*object.String)
	if !ok {
		t.Errorf("object is not *object.String, got=%T (%+v)", evaluated, evaluated)
		return false
	}

	if stringObj.Value != expected {
		t.Errorf("object has the wrong value, expected=%s, got=%s", expected, stringObj.Value)
		return false
	}

	return true
}

func testArrayObject(t *testing.T, evaluated object.Object, expected []int) bool {
	arrayObj, ok := evaluated.(*object.Array)
	if !ok {
		t.Errorf("object is not *object.Array, got=%T (%+v)", evaluated, evaluated)
		return false
	}
	arr := []object.Object(*arrayObj)
	if len(arr) != len(expected) {
		t.Errorf("unequal array lengths, expected=%v, got=%v", expected, arr)
	}
	for i := range arr {
		if !testIntegerObject(t, arr[i], int64(expected[i])) {
			return false
		}
	}
	return true
}

func testHashObject(t *testing.T, evaluated object.Object, expected map[string]object.Object) bool {
	hashObj, ok := evaluated.(*object.Hash)
	if !ok {
		t.Errorf("object is not *object.Hash, got=%T (%+v)", evaluated, evaluated)
		return false
	}
	m := map[object.HashKey]object.Object(*hashObj)
	if len(expected) != len(m) {
		t.Errorf("unequal hashmap lengths, expected=%v, got=%v", expected, m)
	}
	for k, o := range expected {
		sKey := &object.String{k}
		eVal, ok := m[sKey.Hash()]
		if !ok {
			t.Errorf("evaluated map is missing key=%q", k)
			return false
		}
		if !testIntegerObject(t, eVal, o.(*object.Integer).Value) {
			return false
		}
	}
	return true
}

func testFunctionObject(t *testing.T, evaluated object.Object, expected expectedFn) bool {
	fn, ok := evaluated.(*object.Function)
	if !ok {
		t.Fatalf("object is not Function. got=%T (%+v)", evaluated, evaluated)
	}

	if len(fn.Parameters) != len(expected.parameters) {
		t.Fatalf("function has wrong number of parameters. expected=%v, got=%v", len(expected.parameters), len(fn.Parameters))
	}

	for i := range fn.Parameters {
		if fn.Parameters[i].String() != expected.parameters[i] {
			t.Fatalf("parameter identifier mismatch. expected=%q, got=%q",
				expected.parameters[i], fn.Parameters[i].String())
		}
	}

	if fn.Body.String() != expected.body {
		t.Fatalf("body is not %q. got=%q", expected.body, fn.Body.String())
	}
	return true
}

func testErrorObject(t *testing.T, evaluated object.Object, expected *object.Error) bool {
	errObj, ok := evaluated.(*object.Error)
	if !ok {
		t.Errorf("object is not *object.Error, got=%T (%+v)", evaluated, evaluated)
		return false
	}

	if errObj.Message != expected.Message {
		t.Errorf("object has the wrong value, expected=%s, got=%s",
			expected.Message, errObj.Message)
		return false
	}

	return true
}
