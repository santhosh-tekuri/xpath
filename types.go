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

type DataType int

const (
	Unknown DataType = iota
	NodeSet
	String
	Number
	Boolean
)

var resultTypeNames = []string{"node-set", "string", "number", "boolean"}

func (r DataType) String() string {
	return resultTypeNames[r]
}

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
		if math.IsNaN(v) {
			return "NaN"
		}
		if math.IsInf(v, +1) {
			return "Infinity"
		}
		if math.IsInf(v, -1) {
			return "-Infinity"
		}
		if v == math.Trunc(v) {
			return strconv.Itoa(int(v))
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	}
	panic(fmt.Sprintf("%T is not valid xpath data-type", v))
}

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

/************************************************************************/

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

func Node2Number(n dom.Node) float64 {
	return String2Number(Node2String(n))
}

/************************************************************************/

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

func ClarkName(uri, local string) string {
	if uri == "" {
		return local
	}
	return fmt.Sprintf("{%s}%s", uri, local)
}
