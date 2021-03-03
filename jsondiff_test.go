package jsondiff

import (
	"testing"
)

var cases = []struct {
	a      string
	b      string
	result Difference
}{
	{`{"a": 5}`, `["a"]`, NoMatch},
	{`{"a": 5}`, `{"a": 6}`, NoMatch},
	{`{"a": 5}`, `{"a": true}`, NoMatch},
	{`{"a": 5}`, `{"a": 5}`, FullMatch},
	{`{"a": 5}`, `{"a": 5, "b": 6}`, NoMatch},
	{`{"a": 5, "b": 6}`, `{"a": 5}`, SupersetMatch},
	{`{"a": 5, "b": 6}`, `{"b": 6}`, SupersetMatch},
	{`{"a": null}`, `{"a": 1}`, NoMatch},
	{`{"a": null}`, `{"a": null}`, FullMatch},
	{`{"a": "null"}`, `{"a": null}`, NoMatch},
	// Documents that epsilon comparison is not used when it's not enabled in the options.
	{`{"a": 1.0}`, `{"a": 1.0000000000000000000000001}`, NoMatch},
	// Documents an edge case where exponential numbers aren't evaluated if UseFloats is false.
	// These would both evaluate to 100, but they are considered different because their string
	// representation is different.
	{`{"a": 1e2}`, `{"a": 10e1}`, NoMatch},
	{`{"a": 3.1415}`, `{"a": 3.1415}`, FullMatch},
	{`{"a": 4213123123}`, `{"a": "4213123123"}`, NoMatch},
	{`{"a": 4213123123}`, `{"a": 4213123123}`, FullMatch},
}

func TestCompare(t *testing.T) {
	opts := DefaultConsoleOptions()
	opts.PrintTypes = false
	for i, c := range cases {
		result, _ := Compare([]byte(c.a), []byte(c.b), &opts)
		if result != c.result {
			t.Errorf("case %d failed, got: %s, expected: %s", i, result, c.result)
		}
	}
}

func TestCompareFloatsWithEpsilon(t *testing.T) {
	opts := DefaultConsoleOptions()
	opts.PrintTypes = false
	opts.UseFloats = true

	var floatCases = []struct {
		a      string
		b      string
		result Difference
	}{
		{`{"a": 3.1415926535897}`, `{"a": 3.141592653589700000000001}`, FullMatch},
		{`{"a": 3.1415926535897}`, `{"a": 3.1415926535898}`, NoMatch},
		{`{"a": 1}`, `{"a": 1.0000000000000000000000001}`, FullMatch},
		{`{"a": 1.0}`, `{"a": 1.0000000000000000000000001}`, FullMatch},
		// Documents how the scaled epsilon method breaks down when comparing to 0.
		{`{"a": 0.0}`, `{"a": 0.0000000000000000000000000000000000000000000001}`, NoMatch},
		// Exponential notation is parsed when UseFloats is true
		{`{"a": 1e2}`, `{"a": 10e1}`, FullMatch},
	}
	for i, c := range floatCases {
		result, _ := Compare([]byte(c.a), []byte(c.b), &opts)
		if result != c.result {
			t.Errorf("case %d failed, got: %s, expected: %s", i, result, c.result)
		}
	}
}
