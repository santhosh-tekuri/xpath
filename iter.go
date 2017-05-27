// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"github.com/santhosh-tekuri/dom"
)

type iterator interface {
	next() dom.Node
}

var iterators = []func(dom.Node) iterator{
	childAxis,
	descendantAxis,
	parentAxis,
	ancestorAxis,
	followingSiblingAxis,
	precedingSiblingAxis,
	followingAxis,
	precedingAxis,
	attributeAxis,
	namespaceAxis,
	selfAxis,
	descendantOrSelfAxis,
	ancestorOrSelfAxis,
}

type emptyIter struct{}

func (emptyIter) next() dom.Node {
	return nil
}

/************************************************************************/

func selfAxis(n dom.Node) iterator {
	return &onceIter{n}
}

func parentAxis(n dom.Node) iterator {
	return &onceIter{parent(n)}
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

/************************************************************************/

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

/************************************************************************/

func attributeAxis(n dom.Node) iterator {
	if e, ok := n.(*dom.Element); ok {
		return &attrIter{e, 0}
	}
	return emptyIter{}
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

/************************************************************************/

func precedingSiblingAxis(n dom.Node) iterator {
	if p := n.Parent(); p != nil {
		for i, child := range n.Parent().Children() {
			if child == n {
				return &reverseIter{n.Parent().Children(), i - 1}
			}
		}
	}
	return emptyIter{}
}

type reverseIter struct {
	arr []dom.Node
	i   int
}

func (iter *reverseIter) hasNext() bool {
	return iter.i >= 0
}

func (iter *reverseIter) next() dom.Node {
	if iter.i >= 0 {
		n := iter.arr[iter.i]
		iter.i--
		return n
	}
	return nil
}

/************************************************************************/

func ancestorAxis(n dom.Node) iterator {
	return &ancestorOrSelfIter{n.Parent()}
}

func ancestorOrSelfAxis(n dom.Node) iterator {
	return &ancestorOrSelfIter{n}
}

type ancestorOrSelfIter struct {
	n dom.Node
}

func (iter *ancestorOrSelfIter) next() dom.Node {
	if iter.n != nil {
		n := iter.n
		iter.n = parent(n)
		return n
	}
	return nil
}

/************************************************************************/

func descendantAxis(n dom.Node) iterator {
	return &descendantIter{nil, childAxis(n)}
}

func descendantOrSelfAxis(n dom.Node) iterator {
	return &descendantIter{nil, selfAxis(n)}
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

/************************************************************************/

func followingAxis(n dom.Node) iterator {
	return &followingIter{n, followingSiblingAxis(n), emptyIter{}}
}

type followingIter struct {
	contextNode    dom.Node
	siblings       iterator
	currentSibling iterator
}

func (iter *followingIter) next() dom.Node {
	var n dom.Node
	for {
		n = iter.currentSibling.next()
		if n != nil {
			break
		}

		var sibling dom.Node
		for {
			sibling = iter.siblings.next()
			if sibling != nil {
				break
			}

			if _, ok := iter.contextNode.(*dom.Document); ok {
				return nil
			}
			iter.contextNode = parent(iter.contextNode)
			if _, ok := iter.contextNode.(*dom.Document); ok {
				return nil
			}
			iter.siblings = followingSiblingAxis(iter.contextNode)
		}
		iter.currentSibling = descendantOrSelfAxis(sibling)
	}
	return n
}

/************************************************************************/

func precedingAxis(n dom.Node) iterator {
	return &precedingIter{ancestorOrSelfAxis(n), emptyIter{}, &reverseIter{nil, -1}, nil}
}

type precedingIter struct {
	ancestorOrSelf   iterator
	precedingSibling iterator
	childrenOrSelf   *reverseIter
	stack            []*reverseIter
}

func (iter *precedingIter) next() dom.Node {
	for {
		n := iter.childrenOrSelf.next()
		if n == nil {
			if len(iter.stack) == 0 {
				var ps dom.Node
				for {
					ps = iter.precedingSibling.next()
					if ps == nil {
						as := iter.ancestorOrSelf.next()
						if as == nil {
							return nil
						}
						iter.precedingSibling = precedingSiblingAxis(as)
					} else {
						break
					}
				}
				iter.childrenOrSelf = childrenOrSelfIter(ps)
			} else {
				iter.childrenOrSelf = iter.stack[len(iter.stack)-1]
				iter.stack = iter.stack[:len(iter.stack)-1]
			}
			continue
		}
		if iter.childrenOrSelf.hasNext() {
			iter.stack = append(iter.stack, iter.childrenOrSelf)
			iter.childrenOrSelf = childrenOrSelfIter(n)
			continue
		}
		return n
	}
}

func childrenOrSelfIter(n dom.Node) *reverseIter {
	arr := []dom.Node{n}
	children := childAxis(n)
	for {
		c := children.next()
		if c == nil {
			return &reverseIter{arr, len(arr) - 1}
		}
		arr = append(arr, c)
	}
}

/************************************************************************/

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
