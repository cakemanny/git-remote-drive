package store

import (
	"fmt"
	"testing"
)

type fakeIDGetter struct{}

// pretend we have a path like:
// a/b/c
// a, b and c have ids 1, 2 and 3 respectively
func (fakeIDGetter) GetID(name string, parentID string) (string, error) {
	fmt.Printf("name='%s',parentID='%s'\n", name, parentID)
	if name == "a" && parentID == "" {
		return "1", nil
	} else if name == "b" && parentID == "1" {
		return "2", nil
	} else if name == "c" && parentID == "2" {
		return "3", nil
	}
	return "", ErrNotFound
}

func TestGetIDRecursive(t *testing.T) {
	matrix := []struct {
		input    string
		expected string
	}{
		{"a/b/c", "3"},
		{"a/b", "2"},
		{"a", "1"},
	}

	for _, v := range matrix {
		actual, _ := GetIDRecursive(fakeIDGetter{}, v.input)
		if actual != v.expected {
			t.Errorf("GetParentID(%s), expected: %s, actual: %s", v.input, actual, v.expected)
		}
	}

	nonExistantPaths := []string{
		"a/b/d",
		"b/c",
		"x/a/b/c",
	}
	for _, p := range nonExistantPaths {
		actual, err := GetIDRecursive(fakeIDGetter{}, p)
		if err != ErrNotFound {
			t.Errorf("for input %s, expected ErrNotFound, actual (%s,%s)", p, actual, err)
		}
	}
}
