package test

type Bar string

var (
	BarA Bar = "a"
	BarB     = "b" // Not a Bar, unlike iota
	BarC Foo = 25
	BarD Bar = "d"
	BarE     = Bar("e")
)

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
