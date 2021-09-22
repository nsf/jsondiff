package jsondiff

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"
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
	Normal           Tag
	Added            Tag
	Removed          Tag
	Changed          Tag
	Prefix           string
	Indent           string
	PrintTypes       bool
	ChangedSeparator string
	// When provided, this function will be used to compare two numbers. By default numbers are compared using their
	// literal representation byte by byte.
	CompareNumbers func(a, b json.Number) bool
	// When true, only differences will be printed. By default, it will print the full json.
	SkipMatches bool
	Skip        func(path string, a, b interface{}) bool
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
		Added:            Tag{Begin: "\033[0;32m", End: "\033[0m"},
		Removed:          Tag{Begin: "\033[0;31m", End: "\033[0m"},
		Changed:          Tag{Begin: "\033[0;33m", End: "\033[0m"},
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
	buf     bytes.Buffer
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

func (ctx *context) newline(s string) {
	ctx.buf.WriteString(s)
	if ctx.lastTag != nil {
		ctx.buf.WriteString(ctx.lastTag.End)
	}
	ctx.buf.WriteString("\n")
	ctx.buf.WriteString(ctx.opts.Prefix)
	for i := 0; i < ctx.level; i++ {
		ctx.buf.WriteString(ctx.opts.Indent)
	}
	if ctx.lastTag != nil {
		ctx.buf.WriteString(ctx.lastTag.Begin)
	}
}

func (ctx *context) key(k string) {
	ctx.buf.WriteString(strconv.Quote(k))
	ctx.buf.WriteString(": ")
}

func (ctx *context) writeValue(v interface{}, full bool) {
	switch vv := v.(type) {
	case bool:
		ctx.buf.WriteString(strconv.FormatBool(vv))
	case json.Number:
		ctx.buf.WriteString(string(vv))
	case string:
		ctx.buf.WriteString(strconv.Quote(vv))
	case []interface{}:
		if full {
			if len(vv) == 0 {
				ctx.buf.WriteString("[")
			} else {
				ctx.level++
				ctx.newline("[")
			}
			for i, v := range vv {
				ctx.writeValue(v, true)
				if i != len(vv)-1 {
					ctx.newline(",")
				} else {
					ctx.level--
					ctx.newline("")
				}
			}
			ctx.buf.WriteString("]")
		} else {
			ctx.buf.WriteString("[]")
		}
	case map[string]interface{}:
		if full {
			if len(vv) == 0 {
				ctx.buf.WriteString("{")
			} else {
				ctx.level++
				ctx.newline("{")
			}
			i := 0
			for k, v := range vv {
				ctx.key(k)
				ctx.writeValue(v, true)
				if i != len(vv)-1 {
					ctx.newline(",")
				} else {
					ctx.level--
					ctx.newline("")
				}
				i++
			}
			ctx.buf.WriteString("}")
		} else {
			ctx.buf.WriteString("{}")
		}
	default:
		ctx.buf.WriteString("null")
	}

	ctx.writeTypeMaybe(v)
}

func (ctx *context) writeTypeMaybe(v interface{}) {
	if ctx.opts.PrintTypes {
		ctx.buf.WriteString(" ")
		ctx.writeType(v)
	}
}

func (ctx *context) writeType(v interface{}) {
	switch v.(type) {
	case bool:
		ctx.buf.WriteString("(boolean)")
	case json.Number:
		ctx.buf.WriteString("(number)")
	case string:
		ctx.buf.WriteString("(string)")
	case []interface{}:
		ctx.buf.WriteString("(array)")
	case map[string]interface{}:
		ctx.buf.WriteString("(object)")
	default:
		ctx.buf.WriteString("(null)")
	}
}

func (ctx *context) writeMismatch(a, b interface{}) {
	ctx.writeValue(a, false)
	ctx.buf.WriteString(ctx.opts.ChangedSeparator)
	ctx.writeValue(b, false)
}

func (ctx *context) tag(tag *Tag) {
	if ctx.lastTag == tag {
		return
	} else if ctx.lastTag != nil {
		ctx.buf.WriteString(ctx.lastTag.End)
	}
	ctx.buf.WriteString(tag.Begin)
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

func (ctx *context) printMismatch(a, b interface{}) {
	ctx.tag(&ctx.opts.Changed)
	ctx.writeMismatch(a, b)
}

func (ctx *context) shouldSkip(path string, a, b interface{}) bool {
	if path != "" && ctx.opts.Skip != nil {
		// Remove . for root level.
		path = strings.TrimLeft(path, ".")
		return ctx.opts.Skip(path, a, b)
	}
	return false
}

func (ctx *context) printDiff(path string, a, b interface{}, beforePrint func()) bool {
	if ctx.shouldSkip(path, a, b) {
		return false
	}

	gotDifference := false

	if a == nil || b == nil {
		if a == nil && b == nil {
			if !ctx.opts.SkipMatches {
				beforePrint()
				ctx.tag(&ctx.opts.Normal)
				ctx.writeValue(a, false)
			}
			ctx.result(FullMatch)
			return false
		} else {
			beforePrint()
			ctx.printMismatch(a, b)
			ctx.result(NoMatch)
			return true
		}
	}

	ka := reflect.TypeOf(a).Kind()
	kb := reflect.TypeOf(b).Kind()
	if ka != kb {
		beforePrint()
		ctx.printMismatch(a, b)
		ctx.result(NoMatch)
		return true
	}
	switch ka {
	case reflect.Bool:
		if a.(bool) != b.(bool) {
			beforePrint()
			ctx.printMismatch(a, b)
			ctx.result(NoMatch)
			return true
		}
	case reflect.String:
		switch aa := a.(type) {
		case json.Number:
			bb, ok := b.(json.Number)
			if !ok || !ctx.compareNumbers(aa, bb) {
				beforePrint()
				ctx.printMismatch(a, b)
				ctx.result(NoMatch)
				return true
			}
		case string:
			bb, ok := b.(string)
			if !ok || aa != bb {
				beforePrint()
				ctx.printMismatch(a, b)
				ctx.result(NoMatch)
				return true
			}
		}
	case reflect.Slice:
		sa, sb := a.([]interface{}), b.([]interface{})
		salen, sblen := len(sa), len(sb)
		max := salen
		if sblen > max {
			max = sblen
		}

		if max > 0 {
			ctx.level++
		}

		printedHeader := false
		originalLevel := ctx.level
		writeHeader := func() {
			if printedHeader {
				return
			}

			printedHeader = true
			beforePrint()
			ctx.tag(&ctx.opts.Normal)
			if max == 0 {
				ctx.buf.WriteString("[")
			} else {
				currentLevel := ctx.level
				ctx.level = originalLevel
				ctx.newline("[")
				ctx.level = currentLevel
			}
		}

		if !ctx.opts.SkipMatches {
			writeHeader()
		}

		for i := 0; i < max; i++ {
			hadChanges := false
			if i < salen && i < sblen {
				hadChanges = ctx.printDiff(path, sa[i], sb[i], func() {
					writeHeader()
				})
			} else if i < salen {
				hadChanges = true
				ctx.tag(&ctx.opts.Removed)
				ctx.writeValue(sa[i], true)
				ctx.result(SupersetMatch)
			} else if i < sblen {
				hadChanges = true
				ctx.tag(&ctx.opts.Added)
				ctx.writeValue(sb[i], true)
				ctx.result(NoMatch)
			}

			if i == max-1 {
				ctx.level--
			}

			if hadChanges || !ctx.opts.SkipMatches {
				ctx.tag(&ctx.opts.Normal)
				if i != max-1 {
					ctx.newline(",")
				} else {
					ctx.newline("")
				}
			}

			if hadChanges {
				gotDifference = true
			}
		}

		if gotDifference || !ctx.opts.SkipMatches {
			ctx.buf.WriteString("]")
			ctx.writeTypeMaybe(a)
		}

		return gotDifference
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

		if len(keys) > 0 {
			ctx.level++
		}

		originalLevel := ctx.level
		printedHeader := false
		writeHeader := func() {
			if printedHeader {
				return
			}

			printedHeader = true
			beforePrint()
			ctx.tag(&ctx.opts.Normal)
			if len(keys) == 0 {
				ctx.buf.WriteString("{")
			} else {
				currentLevel := ctx.level
				ctx.level = originalLevel
				ctx.newline("{")
				ctx.level = currentLevel
			}
		}

		if !ctx.opts.SkipMatches {
			writeHeader()
		}

		for i, k := range keys {
			va, aok := ma[k]
			vb, bok := mb[k]
			hadChanges := false
			if aok && bok {
				hadChanges = ctx.printDiff(path + "." + k, va, vb, func() {
					writeHeader()
					ctx.key(k)
				})
			} else if aok {
				writeHeader()
				hadChanges = true
				ctx.tag(&ctx.opts.Removed)
				ctx.key(k)
				ctx.writeValue(va, true)
				ctx.result(SupersetMatch)
			} else if bok {
				writeHeader()
				hadChanges = true
				ctx.tag(&ctx.opts.Added)
				ctx.key(k)
				ctx.writeValue(vb, true)
				ctx.result(NoMatch)
			}

			if i == len(keys)-1 {
				ctx.level--
			}

			if hadChanges || !ctx.opts.SkipMatches {
				ctx.tag(&ctx.opts.Normal)
				if i != len(keys)-1 {
					ctx.newline(",")
				} else {
					ctx.newline("")
				}
			}

			if hadChanges {
				gotDifference = true
			}
		}

		if gotDifference || !ctx.opts.SkipMatches {
			ctx.buf.WriteString("}")
			ctx.writeTypeMaybe(a)
		}

		return gotDifference
	}

	if !ctx.opts.SkipMatches {
		beforePrint()
		ctx.tag(&ctx.opts.Normal)
		ctx.writeValue(a, true)
		ctx.result(FullMatch)
	}

	return gotDifference
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

	ctx := context{opts: opts}
	ctx.printDiff("", av, bv, func() {})
	if ctx.lastTag != nil {
		ctx.buf.WriteString(ctx.lastTag.End)
	}
	return ctx.diff, ctx.buf.String()
}
