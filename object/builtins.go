package object

import "fmt"

var Builtins = []struct{
	Name string
	Builtin
}{
	{
		Name: "len",
		Builtin: Builtin(func(objs []Object) Object {
			if len(objs) != 1 {
				return &Error{"len() takes 1 argument"}
			}

			switch obj := objs[0].(type) {
			case *Array:
				return &Integer{Value: int64(len([]Object(*obj)))}
			case *String:
				return &Integer{Value: int64(len(obj.Value))}
			default:
				return &Error{"len() argument must be iterable"}
			}
		}),
	},
	{
		Name: "puts",
		Builtin: Builtin(func(objs []Object) Object {
			if len(objs) != 1 {
				return &Error{"puts() takes 1 argument"}
			}

			switch obj := objs[0].(type) {
			case *String:
				fmt.Println(obj.Value)
				return nil
			case *Integer:
				fmt.Println(obj.Value)
				return nil
			case *Boolean:
				if obj.Value { fmt.Println("true") } else { fmt.Println("false") }
				return nil
			case *Array:
				fmt.Printf("%v\n", []Object(*obj))
				return nil
			default:
				return &Error{"puts() argument cannot be of type %T"}
			}
		}),
	},
	{
		Name: "push",
		Builtin: Builtin(func(objs []Object) Object {
			if len(objs) < 2 {
				return &Error{"push() takes 2 or more arguments"}
			}
			arr, ok := objs[0].(*Array)
			if !ok {
				return &Error{"first argument to push() must be an array"}
			}
			*arr = append(*arr, objs[1:]...)
			return arr
		}),
	},
	{
		Name: "pop",
		Builtin: Builtin(func(objs []Object) Object {
			if len(objs) != 1 {
				return &Error{"pop() takes 1 argument"}
			}
			arr, ok := objs[0].(*Array)
			if !ok {
				return &Error{"pop() argument must be an array"}
			}
			lastVal := (*arr)[len(*arr) - 1]
			*arr = (*arr)[:len(*arr) - 1]
			return lastVal
		}),
	},
}

func GetBuiltinByName(name string) Builtin {
	for _, def := range Builtins {
		if name == def.Name {
			return def.Builtin
		}
	}
	return nil
}

func NewError(format string, a ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, a...)}
}
