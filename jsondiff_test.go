package jsondiff

import (
	"encoding/json"
	"math"
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
	{`{"a": 3.1415}`, `{"a": 3.14156}`, NoMatch},
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
	epsilon := math.Nextafter(1.0, 2.0) - 1.0

	opts := DefaultConsoleOptions()
	opts.PrintTypes = false
	opts.CompareNumbers = func(an, bn json.Number) bool {
		a, err1 := an.Float64()
		b, err2 := bn.Float64()
		if err1 != nil || err2 != nil {
			// fallback to byte by byte comparison if conversion fails
			return an == bn
		}
		// Scale the epsilon based on the relative size of the numbers being compared.
		// For numbers greater than 2.0, EPSILON will be smaller than the difference between two
		// adjacent floats, so it needs to be scaled up. For numbers smaller than 1.0, EPSILON could
		// easily be larger than the numbers we're comparing and thus needs scaled down. This method
		// could still break down for numbers that are very near 0, but it's the best we can do
		// without knowing the relative scale of such numbers ahead of time.
		var scaledEpsilon = epsilon * math.Max(math.Abs(a), math.Abs(b))
		return math.Abs(a-b) < scaledEpsilon
	}

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
