package test

import "github.com/elliotchance/switch-check/test/another-pkg"

type Baz int64

const (
	// Since we are casting iota the type also carried down to BazB and BazC.
	BazA = Baz(iota)
	BazB
	BazC
)

func allBaz() {
	var b Baz
	switch b {
	case BazA, BazB, BazC, anotherpkg.BazD:
	}
}

func missingSomeBaz() {
	var b Baz
	switch b {
	case BazC:
	}
}
