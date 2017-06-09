// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpath

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/dom"
)

func TestSimplify(t *testing.T) {
	tests := map[string]interface{}{
		`number(concat('1','2','3'))`:   float64(123),
		`boolean('santhosh')`:           true,
		`boolean('')`:                   false,
		`1<'santhosh'`:                  false,
		`'santhosh'<1`:                  false,
		`//employee/name<'santhosh'`:    false,
		`'santhosh'<//employee/name`:    false,
		`'santhosh' or //employee/name`: true,
		`'' and //employee/name`:        false,
		`//employee/name or 'santhosh'`: true,
		`//employee/name and ''`:        false,
	}
	compiler := new(Compiler)
	for xpath, expected := range tests {
		t.Logf("%v -> %v", xpath, expected)
		expr, err := compiler.Compile(xpath)
		if err != nil {
			t.Errorf("FAIL: %s: %v", xpath, err)
			continue
		}
		if !expr.IsStatic() {
			t.Errorf("FAIL: %s: must be static", xpath)
			continue
		}
		actual, err := expr.Eval(nil, nil)
		if err != nil {
			t.Errorf("FAIL: %s: %v", xpath, err)
			continue
		}
		if actual != expected {
			t.Errorf("FAIL: xpath: %v expected: %v actual: %v", xpath, expected, actual)
		}
	}
}

func TestEval(t *testing.T) {
	functions := FunctionMap{
		"repeat": &Function{String, Args{Mandatory(String), Mandatory(Number)}, CompileFunc(repeat)},
	}
	data, err := ioutil.ReadFile("testdata/tests.json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for file, contexts := range m {
		t.Log(file)
		f, err := os.Open("testdata/files/" + file)
		if err != nil {
			t.Error("FAIL:", err)
			continue
		}
		doc, err := dom.Unmarshal(xml.NewDecoder(f))
		if err != nil {
			t.Error("FAIL:", err)
			continue
		}

		contexts := contexts.(map[string]interface{})
	Loop:
		for contextStr, v := range contexts {
			v := v.(map[string]interface{})

			prefix2uri := make(map[string]string)
			uri2prefix := make(map[string]string)
			if namespaces, ok := v["namespaces"]; ok {
				namespaces := namespaces.(map[string]interface{})
				for prefix, uri := range namespaces {
					uri := uri.(string)
					prefix2uri[prefix] = uri
					curPrefix, ok := uri2prefix[uri]
					if ok {
						if prefix < curPrefix {
							uri2prefix[uri] = prefix
						}
					} else {
						uri2prefix[uri] = prefix
					}
				}
			}

			vars := make(VariableMap)
			if m, ok := v["variables"]; ok {
				m := m.(map[string]interface{})
				for qname, val := range m {
					colon := strings.IndexByte(qname, ':')
					var clarkName string
					if colon == -1 {
						clarkName = qname
					} else {
						prefix := qname[:colon]
						uri, ok := prefix2uri[prefix]
						if !ok {
							t.Errorf("FAIL: unresolved prefix %s", prefix)
							continue Loop
						}
						clarkName = ClarkName(uri, qname[colon+1:])
					}
					vars[clarkName] = val
				}
			}

			compiler := &Compiler{Namespaces: prefix2uri, Functions: functions}
			t.Log(" ", contextStr)
			contextExpr, err := compiler.Compile(contextStr)
			if err != nil {
				t.Error("FAIL:", err)
				continue
			}
			r, err := contextExpr.Eval(doc, vars)
			if err != nil {
				t.Error("FAIL:", err)
				continue
			}
			contextNS := r.([]dom.Node)
			if len(contextNS) != 1 {
				t.Errorf("FAIL: context resulted %d nodes", len(contextNS))
			}
			context := contextNS[0]

			xpaths := v["xpaths"].(map[string]interface{})
			for xpathStr, expected := range xpaths {
				t.Log("    ", xpathStr)

				xpath, err := compiler.Compile(xpathStr)
				if err != nil {
					t.Error("FAIL:", err)
					continue
				}

				got, err := xpath.Eval(context, vars)
				if err != nil {
					t.Error("FAIL:", err)
					continue
				}
				switch expected := expected.(type) {
				case float64:
					_, ok := got.(float64)
					if !ok {
						t.Errorf("FAIL: type mismatch. expected %T, but got %T", expected, got)
					} else if got != expected {
						t.Errorf("FAIL: expected %#v, but got %#v", expected, got)
					}
				case string:
					_, ok := got.(string)
					if !ok {
						t.Errorf("FAIL: type mismatch. expected %T, but got %T", expected, got)
					} else if got != expected {
						t.Errorf("FAIL: expected %#v, but got %#v", expected, got)
					}
				case bool:
					_, ok := got.(bool)
					if !ok {
						t.Errorf("FAIL: type mismatch. expected %T, but got %T", expected, got)
					} else if got != expected {
						t.Errorf("FAIL: expected %#v, but got %#v", expected, got)
					}
				case []interface{}:
					_, ok := got.([]dom.Node)
					if !ok {
						t.Errorf("FAIL: type mismatch. expected []dom.Node, but got %T", got)
					}
					var nodeset []string
					for _, node := range got.([]dom.Node) {
						nodeset = append(nodeset, getXPath(node, uri2prefix))
					}
					log := false
					if len(nodeset) != len(expected) {
						t.Errorf("FAIL: expected %d nodes, but got %d nodes", len(expected), len(nodeset))
						log = true
					} else {
						for i, xpath := range expected {
							if xpath != nodeset[i] {
								t.Errorf("FAIL: node at %d does not match", i)
								log = true
								break
							}
						}
					}
					if log {
						t.Log("Expected:")
						for i, s := range expected {
							t.Logf("  %2d: %v\n", i, s)
						}
						t.Log("Got:")
						for i, s := range nodeset {
							t.Logf("  %2d: %v\n", i, s)
						}
						t.Log("")
					}
				default:
					t.Errorf("FAIL: comparing results of type %T is not implemented", v)
				}
			}
		}
	}
}

func getXPath(n dom.Node, uri2prefix map[string]string) string {
	if _, ok := n.(*dom.Document); ok {
		return "/"
	}

	var arr []string
	for {
		switch x := n.(type) {
		case *dom.Document:
			// do nothing
		case *dom.Element:
			pos := 0
			for _, c := range x.Parent().Children() {
				if c, ok := c.(*dom.Element); ok {
					if c.URI == x.URI && c.Local == x.Local {
						pos++
					}
					if c == x {
						break
					}
				}
			}
			arr = append(arr, fmt.Sprintf("%s[%d]", getQName(x.Name, uri2prefix), pos))
		case *dom.Attr:
			arr = append(arr, "@"+getQName(x.Name, uri2prefix))
		case *dom.Text:
			pos := 0
			for _, c := range x.Parent().Children() {
				if c, ok := c.(*dom.Text); ok {
					pos++
					if c == x {
						break
					}
				}
			}
			arr = append(arr, fmt.Sprintf("text()[%d]", pos))
		case *dom.Comment:
			pos := 0
			for _, c := range x.Parent().Children() {
				if c, ok := c.(*dom.Comment); ok {
					pos++
					if c == x {
						break
					}
				}
			}
			arr = append(arr, fmt.Sprintf("comment()[%d]", pos))
		case *dom.ProcInst:
			pos := 0
			for _, c := range x.Parent().Children() {
				if c, ok := c.(*dom.ProcInst); ok {
					if c.Target == x.Target {
						pos++
					}
					if c == x {
						break
					}
				}
			}
			arr = append(arr, fmt.Sprintf("processing-instruction(%q)[%d]", x.Target, pos))
		case *dom.NameSpace:
			arr = append(arr, fmt.Sprintf("namespace::%s", x.Prefix))
		default:
			panic(fmt.Sprintf("getXPath(%T) is not implemented", x))
		}
		n = Parent(n)
		if n == nil {
			break
		}
	}

	xpath := ""
	for i := len(arr) - 1; i >= 0; i-- {
		xpath += "/" + arr[i]
	}
	return xpath
}

func getQName(name *dom.Name, uri2prefix map[string]string) string {
	if name.URI == "" {
		return name.Local
	}
	prefix, ok := uri2prefix[name.URI]
	if !ok {
		panic(fmt.Sprintf("no prefix bound for %q", name.URI))
	}
	if prefix == "" {
		return name.Local
	}
	return fmt.Sprintf("%s:%s", prefix, name.Local)
}

func repeat(args []interface{}) interface{} {
	return strings.Repeat(args[0].(string), int(args[1].(float64)))
}
