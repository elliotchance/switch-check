package test

type Bar string

var (
	BarA Bar = "a"
	BarB     = "b" // Not a Bar, unlike iota
	BarC Foo = 25
	BarD Bar = "d"
	BarE     = Bar("e")
)

// BarF is too complex to understand. More importantly because there are no
// arguments we shouldn't think NewBar is a type.
var (
	BarF = NewBar()
	BarG = NewBar("a", 123)
)

func NewBar(_ ...interface{}) Bar {
	return BarA
}

func hasAllBars() {
	var bar Bar
	switch bar {
	case BarA, BarD, BarE:
	}
}

func funcLit() {
	a := func() {
		var bar Bar
		switch bar {
		case BarA, BarD:
		}
	}
	a()
}
