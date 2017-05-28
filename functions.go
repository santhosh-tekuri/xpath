// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

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
	"string":           {String, []DataType{NodeSet}, 0, false, nil},
	"name":             {String, []DataType{NodeSet}, 0, false, nil},
	"local-name":       {String, []DataType{NodeSet}, 0, false, nil},
	"namespace-uri":    {String, []DataType{NodeSet}, 0, false, nil},
	"position":         {Number, nil, 0, false, nil},
	"count":            {Number, []DataType{NodeSet}, 1, false, nil},
	"sum":              {Number, []DataType{NodeSet}, 1, false, nil},
	"normalize-space":  {String, []DataType{String}, 0, false, nil},
	"string-length":    {Number, []DataType{String}, 0, false, nil},
	"starts-with":      {Boolean, []DataType{String, String}, 2, false, nil},
	"ends-with":        {Boolean, []DataType{String, String}, 2, false, nil},
	"contains":         {Boolean, []DataType{String, String}, 2, false, nil},
	"concat":           {String, []DataType{String, String}, 2, true, nil},
	"translate":        {String, []DataType{String, String, String}, 3, false, nil},
	"substring":        {String, []DataType{String, Number, Number}, 2, false, nil},
	"substring-before": {String, []DataType{String, String}, 2, false, nil},
	"substring-after":  {String, []DataType{String, String}, 2, false, nil},
	"not":              {Boolean, []DataType{Boolean}, 1, false, nil},
}
