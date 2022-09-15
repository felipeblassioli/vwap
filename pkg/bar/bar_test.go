package bar

import "testing"

func TestBar(t *testing.T) {
	r := Bar()

	if r != "bar" {
		t.Fail()
	}
}
