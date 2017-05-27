// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"sort"
	"strings"

	"github.com/santhosh-tekuri/dom"
)

func reverse(ns []dom.Node) {
	i, j := 0, len(ns)-1
	for i < j {
		ns[i], ns[j] = ns[j], ns[i]
		i++
		j--
	}
}
func order(ns []dom.Node) {
	sort.Slice(ns, func(i, j int) bool {
		return cmp(ns[i], ns[j]) < 0
	})
}

func cmp(n1, n2 dom.Node) int {
	if n1 == n2 {
		return 0
	}
	if !isChild(n1) && !isChild(n2) {
		p1, p2 := parent(n1), parent(n2)
		if p1 == p2 {
			if isNamespace(n1) {
				if isAttribute(n2) {
					return -1
				}
				return strings.Compare(n1.(*dom.NameSpace).Prefix, n2.(*dom.NameSpace).Prefix)
			} else {
				if isNamespace(n2) {
					return 1
				}
				return strings.Compare(n1.(*dom.Attr).Name.String(), n2.(*dom.Attr).Name.String())
			}
		}
		return cmp(p1, p2)
	}

	d1, d2 := depth(n1), depth(n2)
	a1, a2 := n1, n2
	for d1 > d2 {
		a1 = parent(a1)
		d1--
	}
	if a1 == n2 {
		return 1
	}
	for d2 > d1 {
		a2 = parent(a2)
		d2--
	}
	if a2 == n1 {
		return -1
	}

	// a1 and a2 are now at same depth; and are not the same
	for {
		p1, p2 := parent(a1), parent(a2)
		if p1 == p2 {
			return cmpSiblings(a1, a2)
		}
		a1, a2 = p1, p2
	}
}

func cmpSiblings(s1, s2 dom.Node) int {
	// attributes and namespaces sort before child nodes
	if !isChild(s1) {
		return 1
	} else if !isChild(s2) {
		return -1
	}
	iter := followingSiblingAxis(s1)
	for {
		n := iter.next()
		if n == nil {
			break
		}
		if n == s2 {
			return -1
		}
	}
	return 1
}

func isChild(n dom.Node) bool {
	switch n.(type) {
	case *dom.Attr, *dom.NameSpace:
		return false
	default:
		return true
	}
}

func isNamespace(n dom.Node) bool {
	_, ok := n.(*dom.NameSpace)
	return ok
}

func isAttribute(n dom.Node) bool {
	_, ok := n.(*dom.Attr)
	return ok
}

func depth(n dom.Node) int {
	d := 0
	for {
		n = parent(n)
		if n == nil {
			break
		}
		d++
	}
	return d
}
