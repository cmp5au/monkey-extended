# monkey-extended 
<img src="monkey_logo_contrast.png" width="100" height="100">

[monkey website](https://monkeylang.org)

## Usage
```
go install github.com/cmp5au/monkey-extended@latest
$(go env GOPATH)/bin/monkey-extended <options>
```

### Options
- monkey-extended [(-e | --engine) <engine>]
	- start REPL using the desired engine ("vm" or "evaluator")
- monkey-extended [(-e | --engine) <engine>] [(-o | --out) <outfile>] <monkeyfile>
	- evaluate the input monkeyfile using the engine of choice
	- if -o,--out option is provided, engine must be "vm"
	- if file extension is not .koko, this option is the default unless additional flags are provided
- monkey-extended [-k | --koko] <kokofile>
	- interpret the koko bytecode and run the program within
	- if file extension is .koko, this option is the default unless additional flags are provided


## Added Behavior
- `for` loops
    - `break` and `continue` control flow statements
- variable declaration without assignment (ex: `let x;`)
- variable assignment without `let` for declared variables (ex: `x = 1;`)
- deque methods for Array type: `push`, `pop`, `pushleft`, `popleft`
- `delete` builtin for removal from Array or Hash types
- String operators: concatenation with +, comparators (==, !=, <, >, <=, >=), indexing
- `null` literal
- bytecode (de)serialization
- support for running Monkey scripts
- support for "less/greater than or equal to" operators <=, >=
- compiler bugfixes in cases where last statement is not an ExpressionStatement

## Simplified overview diagram
![simplified overview diagram](overview_diagram.png)
