package main

import (
	"fmt"
	"os"
	"os/user"

	// "monkey/compiler"
	// "monkey/lexer"
	// "monkey/parser"
	"monkey/repl"
	// "monkey/serializer"
	// "monkey/vm"
)

func main() {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
 
	fmt.Printf("Hello %s! This is an interactive REPL for the Monkey programming language.\n", user.Username)
	fmt.Println("Feel free to type in commands below.")
	repl.StartInterpretedRepl(os.Stdin, os.Stdout)

	// fmt.Println(os.Args)
//	inBuffer, err := os.ReadFile(os.Args[1])
//	if err != nil {
//		panic(err)
//	}
//	l := lexer.New(string(inBuffer))
//	p := parser.New(l)
//	program := p.ParseProgram()
//	c := compiler.New()
//	if err := c.Compile(program); err != nil {
//		panic(fmt.Sprintf("compiler error: %s", err))
//	}
//	outFile, err := os.Create("a.koko")
//	if err != nil {
//		panic(err)
//	}
//	_, err = outFile.Write(c.Bytecode().Serialize())
//	if err != nil {
//		panic(err)
//	}
//	outFile.Close()
//	// fmt.Printf("wrote %d bytes\n", outN)
//	
//	bytecodeBuffer, err := os.ReadFile("a.koko")
//	bytecode := &compiler.Bytecode{}
//	inN := bytecode.Deserialize(bytecodeBuffer)
//	if inN != len(bytecodeBuffer) {
//		panic("inN != len(bytecodeBuffer)")
//	}
//	machine := vm.New(bytecode)
//	if err = machine.Run(); err != nil {
//		panic(err)
//	}
//	fmt.Println(machine.LastPoppedStackElem().Inspect())
}
