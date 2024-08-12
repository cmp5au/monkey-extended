package evaluator

import (
	"monkey/ast"
	// "monkey/lexer"
	"monkey/object"
	// "monkey/parser"
	"monkey/token"
)

// singleton values
var (
	TRUE     = &object.Boolean{Value: true}
	FALSE    = &object.Boolean{Value: false}
	NULL     = &object.Null{}
	BREAK    = &object.Break{}
	CONTINUE = &object.Continue{}

	// builtin functions
	builtins = map[string]object.Builtin{
		"len": object.GetBuiltinByName("len"),
		"puts": object.GetBuiltinByName("puts"),
	}
)

func Evaluate(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evaluateProgram(node, env)
	case *ast.BlockStatement:
		return evaluateBlockStatement(node, env)
	case *ast.LetStatement:
		return evaluateLetStatement(node, env)
	case *ast.Identifier:
		return evaluateIdentifier(node, env)
	case *ast.IfExpression:
		return evaluateIfExpression(node, env)
	case *ast.ForStatement:
		return evaluateForStatement(node, env)
	case *ast.ReturnStatement:
		return evaluateReturnStatement(node, env)
	case *ast.BreakStatement:
		return BREAK
	case *ast.ContinueStatement:
		return CONTINUE
	case *ast.ExpressionStatement:
		return Evaluate(node.Expression, env)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}
	case *ast.BooleanLiteral:
		if node.Value {
			return TRUE
		} else {
			return FALSE
		}
	case *ast.FunctionLiteral:
		return &object.Function{Parameters: node.Parameters, Body: node.Body, Env: object.NewEnvironment(env)}
	case *ast.ArrayLiteral:
		return evaluateArrayLiteral(node, env)
	case *ast.HashLiteral:
		return evaluateHashLiteral(node, env)
	case *ast.IndexAccess:
		return evaluateIndexAccess(node, env)
	case *ast.BuiltinFunction:
		return evaluateBuiltinFunction(node)
	case *ast.CallExpression:
		return evaluateCallExpression(node, env)
	case *ast.PrefixUnaryOp:
		right := Evaluate(node.Rhs, env)
		if isError(right) {
			return right
		}
		return evaluatePrefixExpression(node.Operator, right)
	case *ast.InfixBinaryOp:
		lhs := Evaluate(node.Lhs, env)
		if isError(lhs) {
			return lhs
		}
		rhs := Evaluate(node.Rhs, env)
		if isError(rhs) {
			return rhs
		}
		return evaluateInfixExpression(node.Operator, lhs, rhs)
	default:
		return object.NewError("unknown node type to evaluate: %T %s", node, node.String())
	}
}

func evaluateProgram(program *ast.Program, env *object.Environment) object.Object {
	var obj object.Object

	for _, stmt := range program.Statements {
		obj = Evaluate(stmt, env)

		switch obj := obj.(type) {
		case *object.ReturnValue:
			return obj.Value
		case *object.Error:
			return obj
		}
	}

	return obj
}

func evaluateBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var obj object.Object
	for _, stmt := range block.Statements {
		obj = Evaluate(stmt, env)

		if obj != nil {
			switch obj.Type() {
			case object.RETURN, object.ERROR, object.BREAK, object.CONTINUE:
				return obj
			}
		}
	}
	return obj
}

func evaluateLetStatement(letStmt *ast.LetStatement, env *object.Environment) object.Object {
	obj := Evaluate(letStmt.Rhs, env)
	if isError(obj) {
		return obj
	}
	env.Set(letStmt.Identifier.Value, obj, true)
	return NULL
}

func evaluateIdentifier(id *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(id.Value); !ok {
		return object.NewError("identifier not found: %s", id.Value)
	} else {
		return val
	}
}

func evaluateReturnStatement(retStmt *ast.ReturnStatement, env *object.Environment) object.Object {
	returnValue := Evaluate(retStmt.ReturnValue, env)
	if isError(returnValue) {
		return returnValue
	}
	return &object.ReturnValue{Value: returnValue}
}

func evaluateArrayLiteral(arr *ast.ArrayLiteral, env *object.Environment) object.Object {
	objectContents := []object.Object{}
	for _, item := range arr.Contents {
		objectContents = append(objectContents, Evaluate(item, env))
	}
	return object.Array(objectContents)
}

func evaluateHashLiteral(hash *ast.HashLiteral, env *object.Environment) object.Object {
	hashMap := map[object.HashKey]object.Object{}
	for _, hashPair := range hash.Contents {
		keyObj := Evaluate(hashPair.Key, env)
		if key, ok := keyObj.(object.Hashable); ok {
			hashMap[key.Hash()] = Evaluate(hashPair.Value, env)
		} else {
			return object.NewError("non-hashable literal key. got=%T (%+v)", key, key)
		}
	}
	return object.Hash(hashMap)
}

func evaluateBuiltinFunction(bf *ast.BuiltinFunction) object.Object {
	obj := object.ExposeBuiltin(bf)
	if obj.Type() == object.NULL {
		return NULL
	}
	return obj
}

func evaluateCallExpression(callExpr *ast.CallExpression, env *object.Environment) object.Object {
	fn := Evaluate(callExpr.Function, env)
	switch fn := fn.(type) {
	case *object.Function:
		if fn == nil {
			id := "UNKNOWN"
			if ident, ok := callExpr.Function.(*ast.Identifier); ok {
				id = ident.Value
			}
			return object.NewError("identifier not found: %s", id)
		}
		callEnv := object.NewEnvironment(fn.Env)
		if len(callExpr.Arguments) != len(fn.Parameters) {
			return object.NewError("incorrect number of parameters: need %d, got %d", len(fn.Parameters), len(callExpr.Arguments))
		}
		for i := range callExpr.Arguments {
			callEnv.Set(fn.Parameters[i].Value, Evaluate(callExpr.Arguments[i], env), true)
		}
		returnedObj := Evaluate(fn.Body, callEnv)
		if returnValue, ok := returnedObj.(*object.ReturnValue); ok {
			return returnValue.Value
		}
		return returnedObj
	case object.Builtin:
		callArgs := []object.Object{}
		for _, arg := range callExpr.Arguments {
			callArgs = append(callArgs, Evaluate(arg, env))
		}
		if result := fn(callArgs); result != nil {
			return result
		}
		return NULL
	default:
		return object.NewError("attempted function call from a non-function expression: %s", fn.Inspect())
	}
}

func evaluateIndexAccess(idxAccess *ast.IndexAccess, env *object.Environment) object.Object {
	switch container := Evaluate(idxAccess.Container, env).(type) {
	case object.Array:
		idxObj := Evaluate(idxAccess.Index, env)
		idx, ok := idxObj.(*object.Integer)
		if !ok {
			return object.NewError("arrays may only be indexed with integer values. got=%T (%+v)",
				idxObj, idxObj)
		}
		if idx.Value >= 0 && idx.Value < int64(len(container)) {
			return container[idx.Value]
		} else if idx.Value < 0 && idx.Value >= int64(-1*len(container)) {
			return container[idx.Value+int64(len(container))]
		}
		return object.NewError("index error: %d is out of bounds for an array of length %d",
			idx.Value, len(container))
	case object.Hash:
		idxObj := Evaluate(idxAccess.Index, env)
		idx, ok := idxObj.(object.Hashable)
		if !ok {
			return object.NewError("index is not hashable. got=%T (%+v)",
				idxObj, idxObj)
		}
		if val, ok := container[idx.Hash()]; ok {
			return val
		}
	}
	return NULL
}

func evaluatePrefixExpression(operator string, rhs object.Object) object.Object {
	switch operator {
	case "!":
		return evaluateBangOperatorExpression(rhs)
	case "-":
		return evaluatePrefixMinusOperatorExpression(rhs)
	default:
		return object.NewError("unknown operator: %s%s", operator, rhs.Type())
	}
}

func evaluateInfixExpression(operator string, lhs, rhs object.Object) object.Object {
	switch {
	case lhs.Type() == object.INTEGER && rhs.Type() == object.INTEGER:
		return evaluateIntegerInfixExpression(operator, lhs, rhs)
	case lhs.Type() == object.BOOLEAN && rhs.Type() == object.BOOLEAN:
		return evaluateBooleanInfixExpression(operator, lhs, rhs)
	case lhs.Type() == object.STRING && rhs.Type() == object.STRING:
		return evaluateStringInfixExpression(operator, lhs, rhs)
	case lhs.Type() != rhs.Type():
		return object.NewError("type mismatch: %s %s %s",
			lhs.Type(), operator, rhs.Type())
	default:
		return object.NewError("unknown operator: %s %s %s",
			lhs.Type(), operator, rhs.Type())
	}
}

func evaluateBangOperatorExpression(rhs object.Object) object.Object {
	rhsBool := castBoolean(rhs)
	if rhsBool.Value {
		return FALSE
	} else {
		return TRUE
	}
}

func evaluatePrefixMinusOperatorExpression(rhs object.Object) object.Object {
	switch rhs := rhs.(type) {
	case *object.Integer:
		rhs.Value = -1 * rhs.Value
		return rhs
	default:
		return object.NewError("unknown operator: -%s", rhs.Type())
	}
}

func castBoolean(obj object.Object) *object.Boolean {
	switch obj := obj.(type) {
	case *object.Integer:
		if obj.Value == 0 {
			return FALSE
		} else {
			return TRUE
		}
	case *object.String:
		if obj.Value == "" {
			return FALSE
		} else {
			return TRUE
		}
	case *object.Boolean:
		return obj
	}
	return FALSE
}

func evaluateIntegerInfixExpression(operator string, lhs, rhs object.Object) object.Object {
	leftValue := lhs.(*object.Integer).Value
	rightValue := rhs.(*object.Integer).Value

	switch operator {
	case token.PLUS:
		return &object.Integer{Value: leftValue + rightValue}
	case token.MINUS:
		return &object.Integer{Value: leftValue - rightValue}
	case token.ASTERISK:
		return &object.Integer{Value: leftValue * rightValue}
	case token.SLASH:
		return &object.Integer{Value: leftValue / rightValue}
	case token.EQ:
		if leftValue == rightValue {
			return TRUE
		} else {
			return FALSE
		}
	case token.NEQ:
		if leftValue != rightValue {
			return TRUE
		} else {
			return FALSE
		}
	case token.LT:
		if leftValue < rightValue {
			return TRUE
		} else {
			return FALSE
		}
	case token.GT:
		if leftValue > rightValue {
			return TRUE
		} else {
			return FALSE
		}
	case token.LTE:
		if leftValue <= rightValue {
			return TRUE
		} else {
			return FALSE
		}
	case token.GTE:
		if leftValue >= rightValue {
			return TRUE
		} else {
			return FALSE
		}
	default:
		return object.NewError("unknown operator: %s %s %s",
			lhs.Type(), operator, rhs.Type())
	}
}

func evaluateBooleanInfixExpression(operator string, lhs, rhs object.Object) object.Object {
	// no need to parse values, can do direct pointer comparison because we have singletons
	// this is a departure from the book, Thorsten adds these cases in evaluateInfixExpression below the integer-specific case
	switch operator {
	case token.EQ:
		if lhs == rhs {
			return TRUE
		} else {
			return FALSE
		}
	case token.NEQ:
		if lhs != rhs {
			return TRUE
		} else {
			return FALSE
		}
	default:
		return object.NewError("unknown operator: %s %s %s",
			lhs.Type(), operator, rhs.Type())
	}
}

func evaluateStringInfixExpression(operator string, lhs, rhs object.Object) object.Object {
	leftValue := lhs.(*object.String).Value
	rightValue := rhs.(*object.String).Value

	switch operator {
	case token.PLUS:
		return &object.String{Value: leftValue + rightValue}
	case token.EQ:
		return &object.Boolean{Value: leftValue == rightValue}
	case token.NEQ:
		return &object.Boolean{Value: leftValue != rightValue}
	case token.LT:
		return &object.Boolean{Value: leftValue < rightValue}
	case token.GT:
		return &object.Boolean{Value: leftValue > rightValue}
	case token.LTE:
		return &object.Boolean{Value: leftValue <= rightValue}
	case token.GTE:
		return &object.Boolean{Value: leftValue >= rightValue}
	default:
		return object.NewError("unknown operator: %s %s %s",
			lhs.Type(), operator, rhs.Type())
	}
}

func evaluateIfExpression(ifExpr *ast.IfExpression, env *object.Environment) object.Object {
	condition := Evaluate(ifExpr.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Evaluate(ifExpr.Consequence, env)
	} else if ifExpr.Alternative != nil {
		return Evaluate(ifExpr.Alternative, env)
	}
	return NULL
}

func evaluateForStatement(forStmt *ast.ForStatement, env *object.Environment) object.Object {
	for {
		condition := Evaluate(forStmt.Condition, env)
		if isError(condition) {
			return condition
		}

		if !isTruthy(condition) {
			return NULL
		}
		bodyEval := Evaluate(forStmt.Body, env)
		if bodyEval == BREAK {
			return NULL
		}
		switch bodyEval.(type) {
		case *object.ReturnValue, *object.Error:
			return bodyEval
		}
	}

	return NULL
}

func isTruthy(obj object.Object) bool {
	return castBoolean(obj) != FALSE
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR
	}
	return false
}
