switch-check
============

`switch-check` is a tool for validating that `switch` statements contain all
enum values.


Installation
------------

```bash
go get -u github.com/elliotchance/switch-check
```

Usage
-----

```bash
switch-check [options] [files or folders...]

  -show-enums
        Show all enums. Useful for debugging.
  -verbose
        Show all files processed.
```


Example
-------

```go
package test

type Foo int

const (
	FooA Foo = iota
	FooB
)

const FooC = Foo(17)
const FooD = FooC
var FooE = Foo(-1)

func fooMissingSomeValues() {
	foo := FooB

	switch foo {
	case FooA:
	case FooC, FooD:
	}
}
```

Run with `switch-check` will output the error:

```
./test/foo.go:33:2 switch is missing cases for: FooB, FooE
```

Known Limitations
-----------------

1. Using expressions to produce enum values are not supported. This level of
type inference requires the compiler (not just the AST). For example this will
not be recognised as a enum value:

```go
var EnumValueA = someFuncThatReturnsAnEnumValue()
```

2. Switch statements must only switch on the value and not contain expressions
in case statements.
