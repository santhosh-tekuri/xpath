// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"github.com/santhosh-tekuri/dom"
)

// A DataType specifies the type of the value a xpath expression evaluates to.
type DataType int

const (
	// Unknown means the actual type cannot be statically determined.
	Unknown DataType = iota

	// NodeSet represents type []dom.Node.
	NodeSet

	// String represents type string.
	String

	// Number represents type float64.
	Number

	// Boolean represents type bool.
	Boolean
)

var resultTypeNames = []string{"unknown", "node-set", "string", "number", "boolean"}

func (r DataType) String() string {
	return resultTypeNames[r]
}

// TypeOf returns xpath type of given value.
func TypeOf(v interface{}) DataType {
	switch v.(type) {
	case []dom.Node:
		return NodeSet
	case string:
		return String
	case float64:
		return Number
	case bool:
		return Boolean
	}
	panic(InvalidValueError{v})
}

/************************************************************************/

// Value2String converts given value to a string.
// The value must be []dom.Node, string, float64 or boolean.
//
// See https://www.w3.org/TR/xpath/#function-string.
func Value2String(v interface{}) string {
	switch v := v.(type) {
	case []dom.Node:
		if len(v) == 0 {
			return ""
		}
		return Node2String(v[0])
	case string:
		return v
	case float64:
		switch {
		case math.IsNaN(v):
			return "NaN"
		case math.IsInf(v, +1):
			return "Infinity"
		case math.IsInf(v, -1):
			return "-Infinity"
		case v == math.Trunc(v):
			return strconv.Itoa(int(v))
		default:
			return strconv.FormatFloat(v, 'f', -1, 64)
		}
	case bool:
		return strconv.FormatBool(v)
	}
	panic(fmt.Sprintf("%T is not valid xpath data-type", v))
}

// Value2Number converts given value to a float64.
// The value must be []dom.Node, string, float64 or boolean.
//
// See https://www.w3.org/TR/xpath/#function-number.
func Value2Number(v interface{}) float64 {
	switch v := v.(type) {
	case []dom.Node:
		return String2Number(Value2String(v))
	case string:
		return String2Number(v)
	case float64:
		return v
	case bool:
		if v {
			return float64(1)
		}
		return float64(0)
	}
	panic(fmt.Sprintf("%T is not valid xpath data-type", v))
}

// Value2Boolean converts given value to a bool.
// The value must be []dom.Node, string, float64 or boolean.
//
// See https://www.w3.org/TR/xpath/#function-boolean.
func Value2Boolean(v interface{}) bool {
	switch v := v.(type) {
	case []dom.Node:
		return len(v) > 0
	case string:
		return len(v) > 0
	case float64:
		return !(v == 0 || math.IsNaN(v))
	case bool:
		return v
	}
	panic(fmt.Sprintf("%T is not valid xpath data-type", v))
}

// Value2Expr returns literal Expr for given value.
// The value must be string, float64 or bool.
func Value2Expr(v interface{}) Expr {
	switch v := v.(type) {
	case string:
		return stringVal(v)
	case float64:
		return numberVal(v)
	case bool:
		return booleanVal(v)
	}
	panic(fmt.Sprintf("%T is not valid literal", v))
}

/************************************************************************/

// Node2String returns string value of the node.
//
// See https://www.w3.org/TR/xpath/#dt-string-value.
func Node2String(n dom.Node) string {
	switch n := n.(type) {
	case *dom.Comment:
		return n.Data
	case *dom.ProcInst:
		return n.Data
	case *dom.Text:
		return n.Data
	case *dom.NameSpace:
		return n.URI
	case *dom.Attr:
		return n.Value
	default:
		buf := new(bytes.Buffer)
		collectText(n, buf)
		return buf.String()
	}
}

// Node2Number returns the string value of the node converted to float64.
func Node2Number(n dom.Node) float64 {
	return String2Number(Node2String(n))
}

/************************************************************************/

// String2Number converts the string value to float64
func String2Number(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}

func collectText(n dom.Node, buf *bytes.Buffer) {
	if t, ok := n.(*dom.Text); ok {
		buf.WriteString(t.Data)
	} else if p, ok := n.(dom.Parent); ok {
		for _, c := range p.Children() {
			collectText(c, buf)
		}
	}
}

// ClarkName returns local if uri is empty, otherwise `{uri}local`.
func ClarkName(uri, local string) string {
	if uri == "" {
		return local
	}
	return fmt.Sprintf("{%s}%s", uri, local)
}
