package main

import (
	"io/ioutil"
	"testing"
)

func TestCLI(t *testing.T) {
	expected, err := ioutil.ReadFile("test/expected.txt")
	if err != nil {
		t.Error(err)
	}

	actual, exitStatus := run(true, true, []string{"."})
	if string(expected) != actual {
		t.Errorf("expected %v, got %v", string(expected), actual)
	}
	if exitStatus != 1 {
		t.Errorf("exit status is %d", exitStatus)
	}
}
