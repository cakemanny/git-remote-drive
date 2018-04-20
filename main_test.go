package main

import (
	"strings"
	"testing"
)

func TestDispatch(t *testing.T) {
	matrix := []struct {
		command, expected string
	}{
		{"capabilities", "push\nfetch\noption\n\n"},
		{"option verbosity 0", "ok\n"},
		{"option verbosity not a number", "err invalid verbosity\n"},
		{"option boobity bob", "unsupported\n"},
	}
	for _, v := range matrix {
		var out strings.Builder
		dispatch(v.command, &out, storeManager{"", fakeStore})
		result := out.String()
		if result != v.expected {
			t.Error("command:", v.command, "expected:", v.expected,
				"actual:", result)
		}
	}
}

func TestListRefs(t *testing.T) {

	expected := "c5d2d737af4b6203aa37ca2ca13476624d11f4ee refs/heads/master\n\n"

	m := storeManager{"", fakeStore}

	var sb strings.Builder
	listRefs(&sb, m)
	result := sb.String()
	if result != expected {
		t.Errorf("command:list\n\texpected:\"%s\",\n\tactual: \"%s\"",
			expected, result)
	}
}
