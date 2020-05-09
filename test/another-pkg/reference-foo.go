// Notice the package name here is different from the directory.

package anotherpkg

import "github.com/elliotchance/switch-check/test"

const BazD = test.Baz(1234)

func fooRefMissingSomeValues() {
	foo := test.FooB

	switch foo {
	case test.FooA:
	case test.FooB:
	}
}
