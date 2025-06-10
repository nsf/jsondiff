package jsondiff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

var compareCases = []struct {
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
	for i, c := range compareCases {
		result, _ := Compare([]byte(c.a), []byte(c.b), &opts)
		if result != c.result {
			t.Errorf("case %d failed, got: %s, expected: %s", i, result, c.result)
		}
	}
}

func TestCompareStreams(t *testing.T) {
	opts := DefaultConsoleOptions()
	opts.PrintTypes = false
	for i, c := range compareCases {
		result, _ := CompareStreams(bytes.NewReader([]byte(c.a)), bytes.NewReader([]byte(c.b)), &opts)
		if result != c.result {
			t.Errorf("case %d failed, got: %s, expected: %s", i, result, c.result)
		}
	}
}

type diffFlag int

const (
	diffSkipMatches diffFlag = 1 << iota
	diffNoSkipString
)

var diffStringCases = []struct {
	a        string
	b        string
	expected string
	flags    diffFlag
}{
	{`{"b":"foo","a":[1,2,3],"c":"zoo","d":"Joe"}`, `{"a":[1,2,4,5],"b":"baz","c":"zoo"}`, `
{
  "a": [
    1,
    2,
    (C:3 => 4:C),
    (A:5:A)
  ],
  "b": (C:"foo" => "baz":C),
  "c": "zoo",
  (R:"d": "Joe":R)
}
	`, 0},
	{`{"a":[{"foo":"bar"},{"b": "c"}]}`, `{"a":[{"foo":"bar"},{"b": "d"}]}`, `
{
  "a": [
    {
      "foo": "bar"
    },
    {
      "b": (C:"c" => "d":C)
    }
  ]
}
	`, 0},
	{`{"b":"foo","a":[1,2,3],"c":"zoo","d":"Joe"}`, `{"a":[1,2,4,5],"b":"baz","c":"zoo"}`, `
{
  "a": [
    (S:[skipped elements:2]:S),
    (C:3 => 4:C),
    (A:5:A)
  ],
  "b": (C:"foo" => "baz":C),
  (S:[skipped keys:1]:S),
  (R:"d": "Joe":R)
}
	`, diffSkipMatches},
	{`{"a":[{"foo":"bar"},{"b": "c"}]}`, `{"a":[{"foo":"bar"},{"b": "d"}]}`, `
{
  "a": [
    (S:[skipped elements:1]:S),
    {
      "b": (C:"c" => "d":C)
    }
  ]
}
	`, diffSkipMatches},
	{`[1,2,3,4,5]`, `[1,3,3,4,5]`, `
[
  (S:[skipped elements:1]:S),
  (C:2 => 3:C),
  (S:[skipped elements:3]:S)
]
	`, diffSkipMatches},
	{`{"a":1,"b":2,"c":3}`, `{"a":1,"b":"foo","c":3}`, `
{
  (S:[skipped keys:1]:S),
  "b": (C:2 => "foo":C),
  (S:[skipped keys:1]:S)
}
	`, diffSkipMatches},
	{`{"a":[1,2,3]}`, `{"b":"foo"}`, `
 {
  (R:"a": [:R)
    (R:1,:R)
    (R:2,:R)
    (R:3:R)
  (R:]:R),
  (A:"b": "foo":A)
}
	`, 0},
	{`{"b":"foo","a":[1,2,3],"c":"zoo","d":"Joe"}`, `{"a":[1,2,4,5],"b":"baz","c":"zoo"}`, `
{
  "a": [
    (C:3 => 4:C),
    (A:5:A)
  ],
  "b": (C:"foo" => "baz":C),
  (R:"d": "Joe":R)
}
	`, diffSkipMatches | diffNoSkipString},
	{`{"a":[{"foo":"bar"},{"b": "c"}]}`, `{"a":[{"foo":"bar"},{"b": "d"}]}`, `
{
  "a": [
    {
      "b": (C:"c" => "d":C)
    }
  ]
}
	`, diffSkipMatches | diffNoSkipString},
	{`[1,2,3,4,5]`, `[1,3,3,4,5]`, `
[
  (C:2 => 3:C)
]
	`, diffSkipMatches | diffNoSkipString},
	{`{"a":1,"b":2,"c":3}`, `{"a":1,"b":"foo","c":3}`, `
{
  "b": (C:2 => "foo":C)
}
	`, diffSkipMatches | diffNoSkipString},
}

func TestDiffString(t *testing.T) {
	opts := DefaultConsoleOptions()
	opts.Added = Tag{Begin: "(A:", End: ":A)"}
	opts.Removed = Tag{Begin: "(R:", End: ":R)"}
	opts.Changed = Tag{Begin: "(C:", End: ":C)"}
	opts.Skipped = Tag{Begin: "(S:", End: ":S)"}
	opts.SkippedObjectProperty = func(n int) string { return fmt.Sprintf("[skipped keys:%d]", n) }
	opts.SkippedArrayElement = func(n int) string { return fmt.Sprintf("[skipped elements:%d]", n) }
	opts.Indent = "  "
	for i, c := range diffStringCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			lopts := opts
			if c.flags&diffSkipMatches != 0 {
				lopts.SkipMatches = true
			}
			if c.flags&diffNoSkipString != 0 {
				lopts.SkippedObjectProperty = nil
				lopts.SkippedArrayElement = nil
			}
			expected := strings.TrimSpace(c.expected)
			_, diff := Compare([]byte(c.a), []byte(c.b), &lopts)
			if diff != expected {
				t.Errorf("got:\n---\n%s\n---\nexpected:\n---\n%s\n---\n", diff, expected)
			}
		})
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

func TestPresenceFeature(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected Difference
	}{
		{
			name:     "presence check with existing value",
			a:        `{"name": "John", "age": 30}`,
			b:        `{"name": "<<PRESENCE>>", "age": "<<PRESENCE>>"}`,
			expected: FullMatch,
		},
		{
			name:     "presence check with missing value",
			a:        `{"name": "John"}`,
			b:        `{"name": "<<PRESENCE>>", "age": "<<PRESENCE>>"}`,
			expected: NoMatch,
		},
		{
			name:     "presence check in array",
			a:        `["value1", "value2", "value3"]`,
			b:        `["<<PRESENCE>>", "<<PRESENCE>>", "<<PRESENCE>>"]`,
			expected: FullMatch,
		},
		{
			name:     "presence check in shorter array",
			a:        `["value1", "value2"]`,
			b:        `["<<PRESENCE>>", "<<PRESENCE>>", "<<PRESENCE>>"]`,
			expected: NoMatch,
		},
		{
			name:     "mixed presence and exact match",
			a:        `{"name": "John", "age": 30, "city": "NYC"}`,
			b:        `{"name": "<<PRESENCE>>", "age": 30, "city": "<<PRESENCE>>"}`,
			expected: FullMatch,
		},
		{
			name:     "mixed presence with mismatch",
			a:        `{"name": "John", "age": 25, "city": "NYC"}`,
			b:        `{"name": "<<PRESENCE>>", "age": 30, "city": "<<PRESENCE>>"}`,
			expected: NoMatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, _ := Compare([]byte(tt.a), []byte(tt.b), nil)
			if diff != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, diff)
			}
		})
	}
}

func TestPresencePartialMatches(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected Difference
	}{
		{
			name:     "superset match - extra properties in a",
			a:        `{"name": "John", "age": 30, "city": "NYC", "country": "USA"}`,
			b:        `{"name": "<<PRESENCE>>", "age": "<<PRESENCE>>"}`,
			expected: SupersetMatch,
		},
		{
			name:     "partial presence with missing required field",
			a:        `{"name": "John", "city": "NYC"}`,
			b:        `{"name": "<<PRESENCE>>", "age": "<<PRESENCE>>", "city": "<<PRESENCE>>"}`,
			expected: NoMatch,
		},
		{
			name:     "nested object with presence check",
			a:        `{"user": {"name": "John", "email": "john@example.com"}, "status": "active"}`,
			b:        `{"user": "<<PRESENCE>>", "status": "<<PRESENCE>>"}`,
			expected: FullMatch,
		},
		{
			name:     "nested object with missing nested field",
			a:        `{"user": {"name": "John"}, "status": "active"}`,
			b:        `{"user": {"name": "<<PRESENCE>>", "email": "<<PRESENCE>>"}, "status": "<<PRESENCE>>"}`,
			expected: NoMatch,
		},
		{
			name:     "array with partial presence",
			a:        `["value1", "value2"]`,
			b:        `["<<PRESENCE>>", "<<PRESENCE>>", "<<PRESENCE>>"]`,
			expected: NoMatch,
		},
		{
			name:     "array superset with presence",
			a:        `["value1", "value2", "value3", "value4"]`,
			b:        `["<<PRESENCE>>", "<<PRESENCE>>"]`,
			expected: SupersetMatch,
		},
		{
			name:     "mixed exact and presence with partial match",
			a:        `{"name": "John", "age": 25, "city": "NYC"}`,
			b:        `{"name": "<<PRESENCE>>", "age": 30, "city": "<<PRESENCE>>", "country": "<<PRESENCE>>"}`,
			expected: NoMatch,
		},
		{
			name:     "complex nested structure with presence",
			a:        `{"users": [{"name": "John"}, {"name": "Jane"}], "count": 2}`,
			b:        `{"users": "<<PRESENCE>>", "count": "<<PRESENCE>>"}`,
			expected: FullMatch,
		},
		{
			name:     "empty object vs presence requirements",
			a:        `{}`,
			b:        `{"required": "<<PRESENCE>>"}`,
			expected: NoMatch,
		},
		{
			name:     "empty array vs presence requirements",
			a:        `[]`,
			b:        `["<<PRESENCE>>"]`,
			expected: NoMatch,
		},
		{
			name:     "null value vs presence requirement",
			a:        `{"field": null}`,
			b:        `{"field": "<<PRESENCE>>"}`,
			expected: FullMatch, // null is considered present
		},
		{
			name:     "boolean false vs presence requirement",
			a:        `{"active": false}`,
			b:        `{"active": "<<PRESENCE>>"}`,
			expected: FullMatch, // false is considered present
		},
		{
			name:     "zero number vs presence requirement",
			a:        `{"count": 0}`,
			b:        `{"count": "<<PRESENCE>>"}`,
			expected: FullMatch, // 0 is considered present
		},
		{
			name:     "empty string vs presence requirement",
			a:        `{"name": ""}`,
			b:        `{"name": "<<PRESENCE>>"}`,
			expected: FullMatch, // empty string is considered present
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, output := Compare([]byte(tt.a), []byte(tt.b), nil)
			if diff != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, diff)
				t.Errorf("Output: %s", output)
			}
		})
	}
}

func TestPresenceEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected Difference
	}{
		{
			name:     "presence check with literal <<PRESENCE>> string",
			a:        `{"message": "<<PRESENCE>>"}`,
			b:        `{"message": "<<PRESENCE>>"}`,
			expected: FullMatch, // both have literal string, should match exactly
		},
		{
			name:     "presence check vs literal string mismatch",
			a:        `{"message": "hello"}`,
			b:        `{"message": "<<PRESENCE>>"}`,
			expected: FullMatch, // presence check should pass
		},
		{
			name:     "literal string vs presence check (reversed)",
			a:        `{"message": "<<PRESENCE>>"}`,
			b:        `{"message": "hello"}`,
			expected: NoMatch, // exact match required, different strings
		},
		{
			name:     "deeply nested presence check",
			a:        `{"level1": {"level2": {"level3": {"value": "exists"}}}}`,
			b:        `{"level1": {"level2": {"level3": {"value": "<<PRESENCE>>"}}}}`,
			expected: FullMatch,
		},
		{
			name:     "presence check in mixed array types",
			a:        `[1, "string", true, {"key": "value"}, [1,2,3]]`,
			b:        `["<<PRESENCE>>", "<<PRESENCE>>", "<<PRESENCE>>", "<<PRESENCE>>", "<<PRESENCE>>"]`,
			expected: FullMatch,
		},
		{
			name:     "presence check with complex object",
			a:        `{"data": {"users": [{"id": 1}, {"id": 2}], "total": 2}}`,
			b:        `{"data": "<<PRESENCE>>"}`,
			expected: FullMatch,
		},
		{
			name:     "multiple presence checks with one missing",
			a:        `{"a": 1, "b": 2}`,
			b:        `{"a": "<<PRESENCE>>", "b": "<<PRESENCE>>", "c": "<<PRESENCE>>"}`,
			expected: NoMatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, output := Compare([]byte(tt.a), []byte(tt.b), nil)
			if diff != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, diff)
				t.Errorf("Output: %s", output)
			}
		})
	}
}
