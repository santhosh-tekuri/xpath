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
		for contextStr, v := range contexts {
			t.Log(" ", contextStr)
			context := getContext(doc, contextStr)
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

			compiler := NewCompiler(prefix2uri)
			xpaths := v["xpaths"].(map[string]interface{})
			for xpathStr, expected := range xpaths {
				t.Log("    ", xpathStr)

				xpath, err := compiler.Compile(xpathStr)
				if err != nil {
					t.Error("FAIL:", err)
					continue
				}

				got := xpath.Eval(context)
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
			arr = append(arr, fmt.Sprintf("%s[%d]", qname(x.Name, uri2prefix), pos))
		case *dom.Attr:
			arr = append(arr, "@"+qname(x.Name, uri2prefix))
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

func qname(name *dom.Name, uri2prefix map[string]string) string {
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
