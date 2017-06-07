// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpath

import (
	"github.com/santhosh-tekuri/dom"
)

// Iterator over collection of dom nodes.
type Iterator interface {
	// Next returns the next element in the iteration.
	// Returns nil if the iteration has no more nodes.
	Next() dom.Node
}

var iterators = []func(dom.Node) Iterator{
	ChildAxis,
	DescendantAxis,
	ParentAxis,
	AncestorAxis,
	FollowingSiblingAxis,
	PrecedingSiblingAxis,
	FollowingAxis,
	PrecedingAxis,
	AttributeAxis,
	NamespaceAxis,
	SelfAxis,
	DescendantOrSelfAxis,
	AncestorOrSelfAxis,
}

type emptyIter struct{}

func (emptyIter) Next() dom.Node {
	return nil
}

/************************************************************************/

// SelfAxis returns Iterator which contains just the context node itself.
//
// This is forward axis.
func SelfAxis(n dom.Node) Iterator {
	return &onceIter{n}
}

// ParentAxis returns Iterator which contains the parent of the context node, if there is one.
//
// This is forward axis.
func ParentAxis(n dom.Node) Iterator {
	return &onceIter{Parent(n)}
}

type onceIter struct {
	n dom.Node
}

func (iter *onceIter) Next() dom.Node {
	if iter.n != nil {
		n := iter.n
		iter.n = nil
		return n
	}
	return nil
}

/************************************************************************/

// ChildAxis returns Iterator which contains the children of the context node.
//
// This is forward axis.
func ChildAxis(n dom.Node) Iterator {
	if p, ok := n.(dom.Parent); ok {
		return &sliceIter{p.Children(), 0}
	}
	return emptyIter{}
}

// FollowingSiblingAxis returns Iterator which contains all the following siblings of the context node.
// If the context node is an attribute node or namespace node, the following-sibling axis is empty.
//
// This is forward axis.
func FollowingSiblingAxis(n dom.Node) Iterator {
	if p := n.Parent(); p != nil {
		for i, child := range p.Children() {
			if n == child {
				return &sliceIter{p.Children(), i + 1}
			}
		}
	}
	return emptyIter{}
}

// NamespaceAxis returns Iterator which contains the namespace nodes of the context node.
// The axis will be empty unless the context node is an element.
//
// This is forward axis.
func NamespaceAxis(n dom.Node) Iterator {
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

func (iter *sliceIter) Next() dom.Node {
	if iter.i < len(iter.arr) {
		n := iter.arr[iter.i]
		iter.i++
		return n
	}
	return nil
}

/************************************************************************/

// AttributeAxis returns Iterator which contains the attributes of the context node.
// The axis will be empty unless the context node is an element.
//
// This is forward axis.
func AttributeAxis(n dom.Node) Iterator {
	if e, ok := n.(*dom.Element); ok {
		return &attrIter{e, 0}
	}
	return emptyIter{}
}

type attrIter struct {
	e *dom.Element
	i int
}

func (iter *attrIter) Next() dom.Node {
	if iter.i < len(iter.e.Attrs) {
		n := iter.e.Attrs[iter.i]
		iter.i++
		return n
	}
	return nil
}

/************************************************************************/

// PrecedingSiblingAxis returns Iterator which contains all the preceding siblings of the context node.
// If the context node is an attribute node or namespace node, the preceding-sibling axis is empty.
//
// This is reverse axis.
func PrecedingSiblingAxis(n dom.Node) Iterator {
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

func (iter *reverseIter) Next() dom.Node {
	if iter.i >= 0 {
		n := iter.arr[iter.i]
		iter.i--
		return n
	}
	return nil
}

/************************************************************************/

// AncestorAxis returns Iterator which contains the ancestors of the context node.
// The ancestors of the context node consist of the parent of context node and
// the parent's parent and so on; thus, the ancestor axis will always include
// the root node, unless the context node is the root node.
//
// This is reverse axis.
func AncestorAxis(n dom.Node) Iterator {
	return &ancestorOrSelfIter{n.Parent()}
}

// AncestorOrSelfAxis returns Iterator which contains the context node and the ancestors of the context node.
// Thus, the ancestor axis will always include the root node.
//
// This is reverse axis.
func AncestorOrSelfAxis(n dom.Node) Iterator {
	return &ancestorOrSelfIter{n}
}

type ancestorOrSelfIter struct {
	n dom.Node
}

func (iter *ancestorOrSelfIter) Next() dom.Node {
	if iter.n != nil {
		n := iter.n
		iter.n = Parent(n)
		return n
	}
	return nil
}

/************************************************************************/

// DescendantAxis returns Iterator which contains the descendants of the context node.
// A descendant is a child or a child of a child and so on.
// Thus the descendant axis never contains attribute or namespace nodes.
//
// This is forward axis.
func DescendantAxis(n dom.Node) Iterator {
	return &descendantIter{nil, ChildAxis(n)}
}

// DescendantOrSelfAxis returns Iterator which contains the context node and the descendants of the context node.
//
// This is forward axis.
func DescendantOrSelfAxis(n dom.Node) Iterator {
	return &descendantIter{nil, SelfAxis(n)}
}

type descendantIter struct {
	stack    []Iterator
	children Iterator
}

func (iter *descendantIter) Next() dom.Node {
	var n dom.Node
	for {
		n = iter.children.Next()
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
	iter.children = ChildAxis(n)
	return n
}

/************************************************************************/

// FollowingAxis returns Iterator which contains all nodes in the same document as the context node
// that are after the context node in document order, excluding any descendants and
// excluding attribute nodes and namespace nodes.
//
// This is forward axis.
func FollowingAxis(n dom.Node) Iterator {
	return &followingIter{AncestorOrSelfAxis(n), emptyIter{}, emptyIter{}}
}

type followingIter struct {
	ancestorOrSelf   Iterator
	followingSibling Iterator // followingSibling of ancestorOrSelf
	descendantOrSelf Iterator // descendantsOrSelf of followingSibling
}

func (iter *followingIter) Next() dom.Node {
	for {
		if n := iter.descendantOrSelf.Next(); n != nil {
			return n
		}
		for {
			if sibling := iter.followingSibling.Next(); sibling != nil {
				iter.descendantOrSelf = DescendantOrSelfAxis(sibling)
				break
			}
			switch ancestor := iter.ancestorOrSelf.Next(); ancestor.(type) {
			case nil, *dom.Document:
				return nil
			default:
				iter.followingSibling = FollowingSiblingAxis(ancestor)
			}
		}
	}
}

/************************************************************************/

// PrecedingAxis returns Iterator which contains all nodes in the same document as the context node
// that are before the context node in document order, excluding any ancestors and
// excluding attribute nodes and namespace nodes.
//
// This is reverse axis.
func PrecedingAxis(n dom.Node) Iterator {
	return &precedingIter{AncestorOrSelfAxis(n), emptyIter{}, &reverseIter{nil, -1}, nil}
}

type precedingIter struct {
	ancestorOrSelf   Iterator
	precedingSibling Iterator
	childrenOrSelf   *reverseIter
	stack            []*reverseIter
}

func (iter *precedingIter) Next() dom.Node {
	for {
		n := iter.childrenOrSelf.Next()
		if n == nil {
			if len(iter.stack) == 0 {
				var ps dom.Node
				for {
					ps = iter.precedingSibling.Next()
					if ps == nil {
						as := iter.ancestorOrSelf.Next()
						if as == nil {
							return nil
						}
						iter.precedingSibling = PrecedingSiblingAxis(as)
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
	children := ChildAxis(n)
	for {
		c := children.Next()
		if c == nil {
			return &reverseIter{arr, len(arr) - 1}
		}
		arr = append(arr, c)
	}
}

/************************************************************************/

// Parent returns parent of given dom.Node as per xpath specification.
//
// """
// unlike dom specification:
// - the element is the parent of each of these attribute nodes.
// - The element is the parent of each of these namespace nodes.
// """
func Parent(n dom.Node) dom.Node {
	switch n := n.(type) {
	case *dom.Attr:
		return n.Owner
	case *dom.NameSpace:
		return n.Owner
	default:
		return n.Parent()
	}
}
