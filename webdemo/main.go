package main

import (
	"github.com/gopherjs/jquery"
	"github.com/nsf/jsondiff"
)

var jq = jquery.NewJQuery

func main() {
	jq("#buttonCompare").On(jquery.CLICK, func(e jquery.Event) {
		a := jq("#jsonLeft").Val()
		b := jq("#jsonRight").Val()
		opts := jsondiff.Options{
			Indent:     "    ",
			Removed:    jsondiff.Tag{Begin: `<span style="background-color: #fd7f7f">`, End: `</span>`},
			Changed:    jsondiff.Tag{Begin: `<span style="background-color: #fcff7f">`, End: `</span>`},
			Added:      jsondiff.Tag{Begin: `<span style="background-color: #8bff7f">`, End: `</span>`},
			PrintTypes: true,
		}
		diff, text := jsondiff.Compare([]byte(a), []byte(b), &opts)
		jq("#resultDiff").SetVal(diff.String())
		jq("#resultText").SetHtml(text)
	})
	jq("#buttonSwap").On(jquery.CLICK, func(e jquery.Event) {
		a := jq("#jsonLeft").Val()
		b := jq("#jsonRight").Val()
		jq("#jsonLeft").SetVal(b)
		jq("#jsonRight").SetVal(a)
	})
	jq("#buttonClear").On(jquery.CLICK, func(e jquery.Event) {
		jq("#jsonLeft").SetVal("")
		jq("#jsonRight").SetVal("")
	})
}
