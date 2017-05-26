// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

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

func TestEval(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/tests.json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for k, v := range m {
		t.Log(k)
		f, err := os.Open("testdata/files/" + k)
		if err != nil {
			t.Error("FAIL:", err)
			continue
		}
		doc, err := dom.Unmarshal(xml.NewDecoder(f))
		if err != nil {
			t.Error("FAIL:", err)
			continue
		}

		v := v.(map[string]interface{})
		ns := map[string]string{
			"":    "",
			"xml": "http://www.w3.org/XML/1998/namespace",
		}
		if _, ok := v["namespaces"]; ok {
			for k, v := range v["namespaces"].(map[string]interface{}) {
				ns[k] = v.(string)
			}
		}

		c := NewCompiler(ns)
		for k, v := range v["contexts"].(map[string]interface{}) {
			t.Log(" ", k)
			context := getContext(doc, k)
			for k, v := range v.(map[string]interface{}) {
				t.Log("    ", k)
				xpath, err := c.Compile(k)
				if err != nil {
					t.Error("FAIL:", err)
					continue
				}
				r := xpath.Eval(context)
				switch v := v.(type) {
				case float64:
					_, ok := r.(float64)
					if !ok {
						t.Errorf("FAIL: type mismatch. expected %T, but got %T", v, r)
					} else if r != v {
						t.Errorf("FAIL: expected %#v, but got %#v", v, r)
					}
				case string:
					_, ok := r.(string)
					if !ok {
						t.Errorf("FAIL: type mismatch. expected %T, but got %T", v, r)
					} else if r != v {
						t.Errorf("FAIL: expected %#v, but got %#v", v, r)
					}
				case bool:
					_, ok := r.(bool)
					if !ok {
						t.Errorf("FAIL: type mismatch. expected %T, but got %T", v, r)
					} else if r != v {
						t.Errorf("FAIL: expected %#v, but got %#v", v, r)
					}
				case []interface{}:
					_, ok := r.([]dom.Node)
					if !ok {
						t.Errorf("FAIL: type mismatch. expected []dom.Node, but got %T", r)
					}
					var nodeset []string
					for _, node := range r.([]dom.Node) {
						nodeset = append(nodeset, getXPath(node, ns))
					}
					log := false
					if len(nodeset) != len(v) {
						t.Errorf("FAIL: expected %d nodes, but got %d nodes", len(v), len(nodeset))
						log = true
					} else {
						for i, xpath := range v {
							if xpath != nodeset[i] {
								t.Errorf("FAIL: node at %d does not match", i)
								log = true
								break
							}
						}
					}
					if log {
						t.Log("Expected:")
						for i, s := range v {
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

func getContext(d *dom.Document, s string) dom.Node {
	if s == "/" {
		return d
	}
	var n dom.Node = d
	for _, tok := range strings.Split(s[1:], "/") {
		if strings.HasPrefix(tok, "@") {
			tok = strings.TrimPrefix(tok, "@")
			colon := strings.IndexByte(tok, ':')
			if colon == -1 {
				e, ok := n.(*dom.Element)
				if !ok {
					panic("attribute in context on non-element")
				}
				n = e.GetAttr("", tok)
				if n == nil {
					panic("cannot find attribute mentioned in context")
				}
			} else {
				panic("context with attribute with namespace is not implemented")
			}
		} else {
			colon := strings.IndexByte(tok, ':')
			if colon == -1 {
				found := false
				for _, c := range n.(dom.Parent).Children() {
					if e, ok := c.(*dom.Element); ok {
						if e.URI == "" && e.Local == tok {
							n = e
							found = true
						}
					}
				}
				if !found {
					panic("no element found for " + tok)
				}
			} else {
				panic("context with elem with namespace is not implemented")
			}
		}
	}
	return n
}

func getXPath(n dom.Node, ns map[string]string) string {
	if _, ok := n.(*dom.Document); ok {
		return "/"
	}

	reverse := make(map[string]string)
	for prefix, uri := range ns {
		curPrefix, ok := reverse[uri]
		if ok {
			if prefix < curPrefix {
				reverse[uri] = prefix
			}
		} else {
			reverse[uri] = prefix
		}
	}
	ns = reverse

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
			arr = append(arr, fmt.Sprintf("%s[%d]", qname(x.Name, ns), pos))
		case *dom.Attr:
			arr = append(arr, qname(x.Name, ns))
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
		default:
			panic(fmt.Sprintf("getXPath(%T) is not implemented", x))
		}
		n = parent(n)
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

func qname(name *dom.Name, ns map[string]string) string {
	prefix, ok := ns[name.URI]
	if !ok {
		panic(fmt.Sprintf("no prefix bound for %q", name.URI))
	}
	if prefix == "" {
		return name.Local
	}
	return fmt.Sprintf("%s:%s", prefix, name.Local)
}
