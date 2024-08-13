package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"monkey/compiler"
	"monkey/evaluator"
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
	"monkey/repl"
	"monkey/vm"
)

const USAGE = `USAGE:
monkey [(-e | --engine) <engine>]
	start REPL using the desired engine
monkey [(-e | --engine) <engine>] [(-o | --out) <outfile>] <monkeyfile>
	evaluate the input monkeyfile using the engine of choice
	if -o,--out option is provided, engine must be "vm"
	if file extension is not .koko, this option is the default unless additional flags are provided
monkey [-k | --koko] <kokofile>
	interpret the koko bytecode and run the program within
	if file extension is .koko, this option is the default unless additional flags are provided`

var (
	engine      string
	outFilePath string
	koko        bool
)

func main() {
	flag.StringVar(&engine, "engine", "vm", "options are \"vm\" or \"evaluator\"")
	flag.StringVar(&engine, "e", "vm", "options are \"vm\" or \"evaluator\" (shorthand)")
	flag.StringVar(&outFilePath, "out", "", "location to write compiled Monkey bytecode")
	flag.StringVar(&outFilePath, "o", "", "location to write compiled Monkey bytecode (shorthand)")
	flag.BoolVar(&koko, "koko", false, "forces program to use VM to run input file contents as if it were Koko bytecode")
	flag.BoolVar(&koko, "k", false, "forces program to use VM to run input file contents as if it were Koko bytecode (shorthand)")
	flag.Parse()
	args := flag.Args()

	// flag validation
	if engine != "vm" && engine != "evaluator" {
		fmt.Printf("options are \"vm\" or \"evaluator\", got=%q\n", engine)
		fmt.Println(USAGE)
		return
	}
	if koko && engine == "evaluator" {
		fmt.Printf("bytecode can only be interpreted using \"vm\" engine\n")
		fmt.Println(USAGE)
		return
	}
	if koko && outFilePath != "" {
		fmt.Printf("bytecode can only be executed, not compiled\n")
		fmt.Println(USAGE)
		return
	}
	if outFilePath != "" && engine != "vm" {
		fmt.Printf("cannot use engine=%q to compile to bytecode and write to file\n", engine)
		fmt.Println(USAGE)
		return
	}

	switch len(args) {
	case 0:
		if koko {
			fmt.Printf("no input file to run as Koko bytecode\n")
			fmt.Println(USAGE)
			return
		}
		if outFilePath != "" {
			fmt.Printf("no input file to transform into Koko bytecode\n")
			fmt.Println(USAGE)
			return
		}
		fmt.Printf("Hello! This is an interactive REPL for the Monkey programming language.\n")
		fmt.Println("Feel free to type in commands below.")
		if engine == "vm" {
			repl.StartCompiledRepl(os.Stdin, os.Stdout)
		} else {
			repl.StartInterpretedRepl(os.Stdin, os.Stdout)
		}
	case 1:
		if strings.HasSuffix(args[0], ".koko") && engine == "vm" && outFilePath == "" {
			koko = true
		}
		if koko {
			kokoBuffer, err := os.ReadFile(args[0])
			bytecode := &compiler.Bytecode{}
			if n := bytecode.Deserialize(kokoBuffer); n != len(kokoBuffer) {
				fmt.Printf("unable to deserialize koko bytecode, could only read %d/%d bytes\n", n, len(kokoBuffer))
				return
			}
			machine := vm.New(bytecode)
			if err = machine.Run(); err != nil {
				fmt.Printf("vm error: %s\n", err)
			}
			fmt.Println(machine.LastPoppedStackElem().Inspect())
			return
		}
		inBuffer, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Printf("could not read file %s: %s\n", args[0], err)
			return
		}
		l := lexer.New(string(inBuffer))
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			fmt.Printf("parser errors:\n")
			for _, err := range p.Errors() {
				fmt.Printf("\t%s\n", err)
			}
			return
		}

		if engine == "evaluator" {
			env := object.NewEnvironment()
			rootNodeEvalObj := evaluator.Evaluate(program, env)
			fmt.Println(rootNodeEvalObj.Inspect())
			return
		}

		c := compiler.New()
		if err := c.Compile(program); err != nil {
			fmt.Printf("compiler error: %s\n", err)
			return
		}

		if outFilePath != "" {
			outFile, err := os.Create(outFilePath)
			if err != nil {
				fmt.Printf("could not create file %s: %s\n", outFilePath, err)
				return
			}
			_, err = outFile.Write(c.Bytecode().Serialize())
			if err != nil {
				fmt.Printf("could not write to file %s: %s\n", outFilePath, err)
				return
			}
			outFile.Close()
			return
		}

		machine := vm.New(c.Bytecode())
		if err = machine.Run(); err != nil {
			fmt.Printf("vm error: %s\n", err)
		}
		fmt.Println(machine.LastPoppedStackElem().Inspect())
	default:
		fmt.Println(USAGE)
		fmt.Printf("invalid command: %q\n", strings.Join(os.Args, " "))
		return
	}
}
