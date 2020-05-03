package test

// Some tests for literals that should not be be included as enum values.

const (
	LiteralA = -1
)

func useLiteralConstant() {
	var i int
	switch i {
	case LiteralA:
	}
}
