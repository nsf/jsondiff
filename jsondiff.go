package jsondiff

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
)

type Difference int

const (
	FullMatch Difference = iota
	SupersetMatch
	NoMatch
	FirstArgIsInvalidJson
	SecondArgIsInvalidJson
	BothArgsAreInvalidJson
)

func (d Difference) String() string {
	switch d {
	case FullMatch:
		return "FullMatch"
	case SupersetMatch:
		return "SupersetMatch"
	case NoMatch:
		return "NoMatch"
	case FirstArgIsInvalidJson:
		return "FirstArgIsInvalidJson"
	case SecondArgIsInvalidJson:
		return "SecondArgIsInvalidJson"
	case BothArgsAreInvalidJson:
		return "BothArgsAreInvalidJson"
	}
	return "Invalid"
}

type Tag struct {
	Begin string
	End   string
}

type Options struct {
	Normal             Tag
	Added              Tag
	Removed            Tag
	Changed            Tag
	Skipped            Tag
	SkippedSliceString func(n int) string
	SkippedKeysString  func(n int) string
	Prefix             string
	Indent             string
	PrintTypes         bool
	ChangedSeparator   string
	// When provided, this function will be used to compare two numbers. By default numbers are compared using their
	// literal representation byte by byte.
	CompareNumbers func(a, b json.Number) bool
	// When true, only differences will be printed. By default, it will print the full json.
	SkipMatches bool
}

// Provides a set of options in JSON format that are fully parseable.
func DefaultJSONOptions() Options {
	return Options{
		Added:            Tag{Begin: "\"prop-added\":{", End: "}"},
		Removed:          Tag{Begin: "\"prop-removed\":{", End: "}"},
		Changed:          Tag{Begin: "{\"changed\":[", End: "]}"},
		ChangedSeparator: ", ",
		Indent:           "    ",
	}
}

// Provides a set of options that are well suited for console output. Options
// use ANSI foreground color escape sequences to highlight changes.
func DefaultConsoleOptions() Options {
	return Options{
		Added:   Tag{Begin: "\033[0;32m", End: "\033[0m"},
		Removed: Tag{Begin: "\033[0;31m", End: "\033[0m"},
		Changed: Tag{Begin: "\033[0;33m", End: "\033[0m"},
		Skipped: Tag{Begin: "\033[0;90m", End: "\033[0m"},
		SkippedSliceString: func(n int) string {
			if n == 1 {
				return "...skipped 1 array element..."
			} else {
				ns := strconv.FormatInt(int64(n), 10)
				return "...skipped " + ns + " array elements..."
			}
		},
		SkippedKeysString: func(n int) string {
			if n == 1 {
				return "...skipped 1 object key..."
			} else {
				ns := strconv.FormatInt(int64(n), 10)
				return "...skipped " + ns + " object keys..."
			}
		},
		ChangedSeparator: " => ",
		Indent:           "    ",
	}
}

// Provides a set of options that are well suited for HTML output. Works best
// inside <pre> tag.
func DefaultHTMLOptions() Options {
	return Options{
		Added:            Tag{Begin: `<span style="background-color: #8bff7f">`, End: `</span>`},
		Removed:          Tag{Begin: `<span style="background-color: #fd7f7f">`, End: `</span>`},
		Changed:          Tag{Begin: `<span style="background-color: #fcff7f">`, End: `</span>`},
		ChangedSeparator: " => ",
		Indent:           "    ",
	}
}

type context struct {
	opts    *Options
	level   int
	lastTag *Tag
	diff    Difference
}

func (ctx *context) compareNumbers(a, b json.Number) bool {
	if ctx.opts.CompareNumbers != nil {
		return ctx.opts.CompareNumbers(a, b)
	} else {
		return a == b
	}
}

func (ctx *context) terminateTag(buf *bytes.Buffer) {
	if ctx.lastTag != nil {
		buf.WriteString(ctx.lastTag.End)
		ctx.lastTag = nil
	}
}

func (ctx *context) newline(buf *bytes.Buffer, s string) {
	buf.WriteString(s)
	ctx.terminateTag(buf)
	buf.WriteString("\n")
	buf.WriteString(ctx.opts.Prefix)
	for i := 0; i < ctx.level; i++ {
		buf.WriteString(ctx.opts.Indent)
	}
	ctx.terminateTag(buf)
}

func (ctx *context) key(buf *bytes.Buffer, k string) {
	buf.WriteString(strconv.Quote(k))
	buf.WriteString(": ")
}

func (ctx *context) writeValue(buf *bytes.Buffer, v interface{}, full bool) {
	switch vv := v.(type) {
	case bool:
		buf.WriteString(strconv.FormatBool(vv))
	case json.Number:
		buf.WriteString(string(vv))
	case string:
		buf.WriteString(strconv.Quote(vv))
	case []interface{}:
		if full {
			if len(vv) == 0 {
				buf.WriteString("[")
			} else {
				ctx.level++
				ctx.newline(buf, "[")
			}
			for i, v := range vv {
				ctx.writeValue(buf, v, true)
				if i != len(vv)-1 {
					ctx.newline(buf, ",")
				} else {
					ctx.level--
					ctx.newline(buf, "")
				}
			}
			buf.WriteString("]")
		} else {
			buf.WriteString("[]")
		}
	case map[string]interface{}:
		if full {
			if len(vv) == 0 {
				buf.WriteString("{")
			} else {
				ctx.level++
				ctx.newline(buf, "{")
			}
			i := 0
			for k, v := range vv {
				ctx.key(buf, k)
				ctx.writeValue(buf, v, true)
				if i != len(vv)-1 {
					ctx.newline(buf, ",")
				} else {
					ctx.level--
					ctx.newline(buf, "")
				}
				i++
			}
			buf.WriteString("}")
		} else {
			buf.WriteString("{}")
		}
	default:
		buf.WriteString("null")
	}

	ctx.writeTypeMaybe(buf, v)
}

func (ctx *context) writeTypeMaybe(buf *bytes.Buffer, v interface{}) {
	if ctx.opts.PrintTypes {
		buf.WriteString(" ")
		ctx.writeType(buf, v)
	}
}

func (ctx *context) writeType(buf *bytes.Buffer, v interface{}) {
	switch v.(type) {
	case bool:
		buf.WriteString("(boolean)")
	case json.Number:
		buf.WriteString("(number)")
	case string:
		buf.WriteString("(string)")
	case []interface{}:
		buf.WriteString("(array)")
	case map[string]interface{}:
		buf.WriteString("(object)")
	default:
		buf.WriteString("(null)")
	}
}

func (ctx *context) writeMismatch(buf *bytes.Buffer, a, b interface{}) {
	ctx.writeValue(buf, a, false)
	buf.WriteString(ctx.opts.ChangedSeparator)
	ctx.writeValue(buf, b, false)
}

func (ctx *context) tag(buf *bytes.Buffer, tag *Tag) {
	if ctx.lastTag == tag {
		return
	} else if ctx.lastTag != nil {
		buf.WriteString(ctx.lastTag.End)
	}
	buf.WriteString(tag.Begin)
	ctx.lastTag = tag
}

func (ctx *context) result(d Difference) {
	if d == NoMatch {
		ctx.diff = NoMatch
	} else if d == SupersetMatch && ctx.diff != NoMatch {
		ctx.diff = SupersetMatch
	} else if ctx.diff != NoMatch && ctx.diff != SupersetMatch {
		ctx.diff = FullMatch
	}
}

func (ctx *context) printMismatch(buf *bytes.Buffer, a, b interface{}) {
	ctx.tag(buf, &ctx.opts.Changed)
	ctx.writeMismatch(buf, a, b)
}

func (ctx *context) printSkipped(buf *bytes.Buffer, n *int, strfunc func(n int) string, last bool) {
	if *n == 0 {
		return
	}
	ctx.tag(buf, &ctx.opts.Skipped)
	buf.WriteString(strfunc(*n))
	if !last {
		ctx.tag(buf, &ctx.opts.Normal)
		ctx.newline(buf, ",")
	}
	*n = 0
}

func (ctx *context) finalize(buf *bytes.Buffer) string {
	ctx.terminateTag(buf)
	return buf.String()
}

func (ctx *context) printDiff(a, b interface{}) string {
	var buf bytes.Buffer

	if a == nil || b == nil {
		// either is nil, means there are just two cases:
		// 1. both are nil => match
		// 2. one of them is nil => mismatch
		if a == nil && b == nil {
			// match
			if !ctx.opts.SkipMatches {
				ctx.tag(&buf, &ctx.opts.Normal)
				ctx.writeValue(&buf, a, false)
				ctx.result(FullMatch)
			}
		} else {
			// mismatch
			ctx.printMismatch(&buf, a, b)
			ctx.result(NoMatch)
		}
		return ctx.finalize(&buf)
	}

	ka := reflect.TypeOf(a).Kind()
	kb := reflect.TypeOf(b).Kind()
	if ka != kb {
		// Go type does not match, this is definitely a mismatch since
		// we parse JSON into interface{}
		ctx.printMismatch(&buf, a, b)
		ctx.result(NoMatch)
		return ctx.finalize(&buf)
	}

	// big switch here handles type-specific mismatches and returns if that's the case
	// buf if control flow goes past through this switch, it's a match
	// NOTE: ka == kb at this point
	switch ka {
	case reflect.Bool:
		if a.(bool) != b.(bool) {
			ctx.printMismatch(&buf, a, b)
			ctx.result(NoMatch)
			return ctx.finalize(&buf)
		}
	case reflect.String:
		// string can be a json.Number here too (because it's a string type)
		switch aa := a.(type) {
		case json.Number:
			bb, ok := b.(json.Number)
			if !ok || !ctx.compareNumbers(aa, bb) {
				ctx.printMismatch(&buf, a, b)
				ctx.result(NoMatch)
				return ctx.finalize(&buf)
			}
		case string:
			bb, ok := b.(string)
			if !ok || aa != bb {
				ctx.printMismatch(&buf, a, b)
				ctx.result(NoMatch)
				return ctx.finalize(&buf)
			}
		}
	case reflect.Slice:
		sa, sb := a.([]interface{}), b.([]interface{})
		salen, sblen := len(sa), len(sb)
		max := salen
		if sblen > max {
			max = sblen
		}
		ctx.tag(&buf, &ctx.opts.Normal)
		if max == 0 {
			buf.WriteString("[")
		} else {
			ctx.level++
			ctx.newline(&buf, "[")
		}

		// did we see any changes between slice elements
		// it's initialized with this expression which means that at least one slice has len() > 0
		// and slice lengths are not equal, in this case we definitely have some changes
		seenDiff := max > 0 && salen != sblen

		// track how many elements we skipped with no changes
		noDiffSpan := 0
		for i := 0; i < max; i++ {
			equals := true
			if i < salen && i < sblen {
				diff := ctx.printDiff(sa[i], sb[i])
				if len(diff) > 0 {
					equals = false
					ctx.printSkipped(&buf, &noDiffSpan, ctx.opts.SkippedSliceString, false)
					buf.WriteString(diff)
				}
			} else if i < salen {
				equals = false
				ctx.printSkipped(&buf, &noDiffSpan, ctx.opts.SkippedSliceString, false)
				ctx.tag(&buf, &ctx.opts.Removed)
				ctx.writeValue(&buf, sa[i], true)
				ctx.result(SupersetMatch)
			} else if i < sblen {
				equals = false
				ctx.printSkipped(&buf, &noDiffSpan, ctx.opts.SkippedSliceString, false)
				ctx.tag(&buf, &ctx.opts.Added)
				ctx.writeValue(&buf, sb[i], true)
				ctx.result(NoMatch)
			}
			if ctx.opts.SkipMatches {
				if equals {
					noDiffSpan++
				} else {
					seenDiff = true
				}
			}
			ctx.tag(&buf, &ctx.opts.Normal)
			if i != max-1 {
				// it was not the last element
				if !ctx.opts.SkipMatches || !equals {
					ctx.newline(&buf, ",")
				}
			} else {
				// it was the last element
				ctx.printSkipped(&buf, &noDiffSpan, ctx.opts.SkippedSliceString, true)
				ctx.level--
				ctx.newline(&buf, "")
			}
		}
		buf.WriteString("]")
		ctx.writeTypeMaybe(&buf, a)

		if ctx.opts.SkipMatches && !seenDiff {
			return ""
		}

		return ctx.finalize(&buf)
	case reflect.Map:
		ma, mb := a.(map[string]interface{}), b.(map[string]interface{})
		keysMap := make(map[string]bool)
		for k := range ma {
			keysMap[k] = true
		}
		for k := range mb {
			keysMap[k] = true
		}
		keys := make([]string, 0, len(keysMap))
		for k := range keysMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ctx.tag(&buf, &ctx.opts.Normal)
		if len(keys) == 0 {
			buf.WriteString("{")
		} else {
			ctx.level++
			ctx.newline(&buf, "{")
		}

		seenDiff := false
		noDiffSpan := 0

		for i, k := range keys {
			va, aok := ma[k]
			vb, bok := mb[k]
			equals := true
			if aok && bok {
				diff := ctx.printDiff(va, vb)
				if len(diff) > 0 {
					equals = false
					ctx.printSkipped(&buf, &noDiffSpan, ctx.opts.SkippedKeysString, false)
					ctx.key(&buf, k)
					buf.WriteString(diff)
				}
			} else if aok {
				equals = false
				ctx.printSkipped(&buf, &noDiffSpan, ctx.opts.SkippedKeysString, false)
				ctx.tag(&buf, &ctx.opts.Removed)
				ctx.key(&buf, k)
				ctx.writeValue(&buf, va, true)
				ctx.result(SupersetMatch)
			} else if bok {
				equals = false
				ctx.printSkipped(&buf, &noDiffSpan, ctx.opts.SkippedKeysString, false)
				ctx.tag(&buf, &ctx.opts.Added)
				ctx.key(&buf, k)
				ctx.writeValue(&buf, vb, true)
				ctx.result(NoMatch)
			}
			if ctx.opts.SkipMatches {
				if equals {
					noDiffSpan++
				} else {
					seenDiff = true
				}
			}

			ctx.tag(&buf, &ctx.opts.Normal)
			if i != len(keys)-1 {
				// it was not the last key
				if !ctx.opts.SkipMatches || !equals {
					ctx.newline(&buf, ",")
				}
			} else {
				// it was the last key
				ctx.printSkipped(&buf, &noDiffSpan, ctx.opts.SkippedKeysString, true)
				ctx.level--
				ctx.newline(&buf, "")
			}
		}
		buf.WriteString("}")
		ctx.writeTypeMaybe(&buf, a)

		if ctx.opts.SkipMatches && !seenDiff {
			return ""
		}

		return ctx.finalize(&buf)
	}
	if !ctx.opts.SkipMatches {
		ctx.tag(&buf, &ctx.opts.Normal)
		ctx.writeValue(&buf, a, true)
		ctx.result(FullMatch)
	}
	return ctx.finalize(&buf)
}

// Compares two JSON documents using given options. Returns difference type and
// a string describing differences.
//
// FullMatch means provided arguments are deeply equal.
//
// SupersetMatch means first argument is a superset of a second argument. In
// this context being a superset means that for each object or array in the
// hierarchy which don't match exactly, it must be a superset of another one.
// For example:
//
//     {"a": 123, "b": 456, "c": [7, 8, 9]}
//
// Is a superset of:
//
//     {"a": 123, "c": [7, 8]}
//
// NoMatch means there is no match.
//
// The rest of the difference types mean that one of or both JSON documents are
// invalid JSON.
//
// Returned string uses a format similar to pretty printed JSON to show the
// human-readable difference between provided JSON documents. It is important
// to understand that returned format is not a valid JSON and is not meant
// to be machine readable.
func Compare(a, b []byte, opts *Options) (Difference, string) {
	var av, bv interface{}
	da := json.NewDecoder(bytes.NewReader(a))
	da.UseNumber()
	db := json.NewDecoder(bytes.NewReader(b))
	db.UseNumber()
	errA := da.Decode(&av)
	errB := db.Decode(&bv)
	if errA != nil && errB != nil {
		return BothArgsAreInvalidJson, "both arguments are invalid json"
	}
	if errA != nil {
		return FirstArgIsInvalidJson, "first argument is invalid json"
	}
	if errB != nil {
		return SecondArgIsInvalidJson, "second argument is invalid json"
	}

	var buf bytes.Buffer

	ctx := context{opts: opts}
	buf.WriteString(ctx.printDiff(av, bv))
	return ctx.diff, buf.String()
}
