// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"github.com/santhosh-tekuri/dom"
)

func alwaysTrue(dom.Node) bool {
	return true
}

func isElement(n dom.Node) bool {
	_, ok := n.(*dom.Element)
	return ok
}

func isComment(n dom.Node) bool {
	_, ok := n.(*dom.Comment)
	return ok
}

func isText(n dom.Node) bool {
	_, ok := n.(*dom.Text)
	return ok
}

func isProcInst(name string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.ProcInst); ok {
			return name == "" || name == n.Target
		}
		return false
	}
}

func testElementNS(uri string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.Element); ok {
			return n.URI == uri
		}
		return false
	}
}

func testElementName(uri, local string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.Element); ok {
			return n.URI == uri && n.Local == local
		}
		return false
	}
}

func testAttrNs(uri string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.Attr); ok {
			return n.URI == uri
		}
		return false
	}
}

func testAttrName(uri, local string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.Attr); ok {
			return n.URI == uri && n.Local == local
		}
		return false
	}
}

func testName(uri, local string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		switch n := n.(type) {
		case *dom.Element:
			return n.URI == uri && n.Local == local
		case *dom.Attr:
			return n.URI == uri && n.Local == local
		case *dom.NameSpace:
			return n.URI == uri
		}
		return false
	}
}
