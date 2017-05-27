// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

type function struct {
	returns   DataType
	args      []DataType
	mandatory int
	variadic  bool
	impl      func(ctx *context, args []interface{}) interface{}
}

func (f *function) canAccept(nArgs int) bool {
	return nArgs >= f.mandatory && (f.variadic || nArgs <= len(f.args))
}

func (f *function) argType(i int) DataType {
	if i < len(f.args) {
		return f.args[i]
	}
	return f.args[len(f.args)-1]
}

var coreFunctions = map[string]*function{
	"starts-with": {Boolean, []DataType{String, String}, 2, false, startsWith},
	"ends-with":   {Boolean, []DataType{String, String}, 2, false, endsWith},
	"contains":    {Boolean, []DataType{String, String}, 2, false, contains},
	"concat":      {String, []DataType{String, String}, 2, true, concat},
	"translate":   {String, []DataType{String, String, String}, 3, false, translate},
}

func startsWith(ctx *context, args []interface{}) interface{} {
	return strings.HasPrefix(args[0].(string), args[1].(string))
}

func endsWith(ctx *context, args []interface{}) interface{} {
	return strings.HasSuffix(args[0].(string), args[1].(string))
}

func contains(ctx *context, args []interface{}) interface{} {
	return strings.Contains(args[0].(string), args[1].(string))
}

func stringLength(ctx *context, args []interface{}) interface{} {
	return utf8.RuneCountInString(args[0].(string))
}

func concat(ctx *context, args []interface{}) interface{} {
	buf := new(bytes.Buffer)
	for _, v := range args {
		buf.WriteString(v.(string))
	}
	return buf.String()
}

func translate(ctx *context, args []interface{}) interface{} {
	from := []rune(args[1].(string))
	to := []rune(args[2].(string))
	replace := make(map[rune]rune)
	remove := make(map[rune]struct{})
	for i, frune := range from {
		if _, ok := replace[frune]; ok {
			continue
		}
		if _, ok := remove[frune]; ok {
			continue
		}
		if i < len(to) {
			replace[frune] = to[i]
		} else {
			remove[frune] = struct{}{}
		}
	}

	str := args[0].(string)
	buf := bytes.NewBuffer(make([]byte, 0, len(str)))
	for _, r := range str {
		if _, ok := remove[r]; ok {
			continue
		}
		if v, ok := replace[r]; ok {
			buf.WriteRune(v)
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
