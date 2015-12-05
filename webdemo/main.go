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
		opts := jsondiff.DefaultHTMLOptions()
		opts.PrintTypes = jq("#checkType").Prop("checked").(bool)
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
