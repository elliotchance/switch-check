package test

import "regexp"

type Foo int

// Define multiple enum values.
const (
	FooA Foo = iota
	FooB
)

// Define a single one.
const FooC = Foo(17)

// Alias.
const FooD = FooC

// Defined as var.
var FooE = Foo(-1)

// Ignored because not of type Foo
const FooF = 123

// This will not be understood as an enum value because of the complex
// expression.
const FooG = FooA + 2

// Make sure "regexp.MustCompile" is not considered a type.
var alnumOrDashRegexp = regexp.MustCompile("[^a-z_0-9-]+")

func ignoredSwitch1() {
	switch {
	case true:
	case FooC == FooD:
	}
}

func fooMissingSomeValues() {
	foo := FooB

	switch foo {
	case FooA:
	case FooC, FooD:
	}
}

func defaultPermitsMissingValues() {
	bar := BarA

	switch bar {
	case BarD:
	case BarE:
	default:
		// This means we don't have to specify all enum values, so be careful
		// using default in these cases.
	}
}
