// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"github.com/santhosh-tekuri/dom"
)

var iterators = []func(dom.Node) iterator{
	childAxis,
	descendantAxis,
	parentAxis,
	ancestorAxis,
	followingSiblingAxis,
	precedingSiblingAxis,
	nil, //followingAxis,
	nil, //precedingAxis,
	attributeAxis,
	namespaceAxis,
	selfAxis,
	descendantOrSelfAxis,
	ancestorOrSelfAxis,
}

func selfAxis(n dom.Node) iterator {
	return &onceIter{n}
}

func parentAxis(n dom.Node) iterator {
	return &onceIter{parent(n)}
}

func attributeAxis(n dom.Node) iterator {
	if e, ok := n.(*dom.Element); ok {
		return &attrIter{e, 0}
	}
	return emptyIter{}
}

func childAxis(n dom.Node) iterator {
	if p, ok := n.(dom.Parent); ok {
		return &sliceIter{p.Children(), 0}
	}
	return emptyIter{}
}

func followingSiblingAxis(n dom.Node) iterator {
	if p := n.Parent(); p != nil {
		for i, child := range p.Children() {
			if n == child {
				return &sliceIter{p.Children(), i + 1}
			}
		}
	}
	return emptyIter{}
}

func precedingSiblingAxis(n dom.Node) iterator {
	if p := n.Parent(); p != nil {
		return &precedingSiblingIter{p, 0, n}
	}
	return emptyIter{}
}

func ancestorAxis(n dom.Node) iterator {
	return &ancestorOrSelfIter{n.Parent()}
}

func ancestorOrSelfAxis(n dom.Node) iterator {
	return &ancestorOrSelfIter{n}
}

func descendantAxis(n dom.Node) iterator {
	return &descendantIter{nil, childAxis(n)}
}

func descendantOrSelfAxis(n dom.Node) iterator {
	return &descendantIter{nil, selfAxis(n)}
}

func namespaceAxis(n dom.Node) iterator {
	if elem, ok := n.(*dom.Element); ok {
		m := make(map[string]struct{})
		ns := []dom.Node{
			&dom.NameSpace{elem, "xml", "http://www.w3.org/XML/1998/namespace"},
		}
		e := elem
		for {
			for prefix, uri := range e.NSDecl {
				if _, ok := m[prefix]; !ok {
					m[prefix] = struct{}{}
					ns = append(ns, &dom.NameSpace{elem, prefix, uri})
				}
			}
			p := e.Parent()
			if p, ok := p.(*dom.Element); ok {
				e = p
			} else {
				break
			}
		}
		return &sliceIter{ns, 0}
	}
	return emptyIter{}
}

type iterator interface {
	next() dom.Node
}

type emptyIter struct{}

func (emptyIter) next() dom.Node {
	return nil
}

type onceIter struct {
	n dom.Node
}

func (iter *onceIter) next() dom.Node {
	if iter.n != nil {
		n := iter.n
		iter.n = nil
		return n
	}
	return nil
}

type sliceIter struct {
	arr []dom.Node
	i   int
}

func (iter *sliceIter) next() dom.Node {
	if iter.i < len(iter.arr) {
		n := iter.arr[iter.i]
		iter.i++
		return n
	}
	return nil
}

type precedingSiblingIter struct {
	p dom.Parent
	i int
	n dom.Node
}

func (iter *precedingSiblingIter) next() dom.Node {
	if iter.i < len(iter.p.Children()) {
		n := iter.p.Children()[iter.i]
		if n != iter.n {
			iter.i++
			return n
		}
	}
	return nil
}

type attrIter struct {
	e *dom.Element
	i int
}

func (iter *attrIter) next() dom.Node {
	if iter.i < len(iter.e.Attrs) {
		n := iter.e.Attrs[iter.i]
		iter.i++
		return n
	}
	return nil
}

type ancestorOrSelfIter struct {
	n dom.Node
}

func (iter *ancestorOrSelfIter) next() dom.Node {
	if iter.n != nil {
		n := iter.n
		iter.n = parent(n)
	}
	return nil
}

type descendantIter struct {
	stack    []iterator
	children iterator
}

func (iter *descendantIter) next() dom.Node {
	var n dom.Node
	for {
		n = iter.children.next()
		if n != nil {
			break
		}
		if len(iter.stack) == 0 {
			return nil
		}
		iter.children = iter.stack[len(iter.stack)-1]
		iter.stack = iter.stack[:len(iter.stack)-1]
	}
	iter.stack = append(iter.stack, iter.children)
	iter.children = childAxis(n)
	return n
}

func parent(n dom.Node) dom.Node {
	switch n := n.(type) {
	case *dom.Attr:
		return n.Owner
	case *dom.NameSpace:
		return n.Owner
	default:
		return n.Parent()
	}
}

func index(n dom.Node) int {
	if n.Parent() == nil {
		return -1
	}
	for i, child := range n.Parent().Children() {
		if child == n {
			return i
		}
	}
	return -1
}
