// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Parse nodes.

package main

import (
	"bytes"
	"fmt"
	"strconv"
	//	"strings"
)

var textFormat = "%s" // Changed to "%q" in tests for better error messages.

// A Node is an element in the parse tree. The interface is trivial.
// The interface contains an unexported method so that only
// types local to this package can satisfy it.
type Node interface {
	Type() NodeType
	String() string
	// Copy does a deep copy of the Node and all its components.
	// To avoid type assertions, some XxxNodes also have specialized
	// CopyXxx methods that return *XxxNode.
	Copy() Node
	Position() Pos // byte position of start of node in full original input string
	// Make sure only functions in this package can create Nodes.
	unexported()
}

// NodeType identifies the type of a parse tree node.
type NodeType int

// Pos represents a byte position in the original input text from which
// this template was parsed.
type Pos int

func (p Pos) Position() Pos {
	return p
}

// unexported keeps Node implementations local to the package.
// All implementations embed Pos, so this takes care of it.
func (Pos) unexported() {
}

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeBool       NodeType = iota // A boolean constant.
	NodeDot                        // The cursor, dot.
	nodeElse                       // An else action. Not added to tree.
	nodeEnd                        // An end action. Not added to tree.
	NodeIdentifier                 // An identifier; always a function name.
	NodeIf                         // An if action.
	NodeList                       // A list of Nodes.
	NodeNil                        // An untyped nil constant.
	NodeNumber                     // A numerical constant.
	NodeString                     // A string constant.
)

// Nodes.

// ListNode holds a sequence of nodes.
type ListNode struct {
	NodeType
	Pos
	Nodes []Node // The element nodes in lexical order.
}

func newListNode(pos Pos) *ListNode {
	return &ListNode{NodeType: NodeList, Pos: pos}
}

func (l *ListNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}

func (l *ListNode) String() string {
	b := new(bytes.Buffer)
	for _, n := range l.Nodes {
		fmt.Fprint(b, n)
	}
	return b.String()
}

func (l *ListNode) CopyList() *ListNode {
	if l == nil {
		return l
	}
	n := newListNode(l.Pos)
	for _, elem := range l.Nodes {
		n.append(elem.Copy())
	}
	return n
}

func (l *ListNode) Copy() Node {
	return l.CopyList()
}

//// TextNode holds plain text.
//type TextNode struct {
//	NodeType
//	Pos
//	Text []byte // The text; may span newlines.
//}
//
//func newText(pos Pos, text string) *TextNode {
//	return &TextNode{NodeType: NodeText, Pos: pos, Text: []byte(text)}
//}
//
//func (t *TextNode) String() string {
//	return fmt.Sprintf(textFormat, t.Text)
//}
//
//func (t *TextNode) Copy() Node {
//	return &TextNode{NodeType: NodeText, Text: append([]byte{}, t.Text...)}
//}

// IdentifierNode holds an identifier.
type IdentifierNode struct {
	NodeType
	Pos
	Ident string // The identifier's name.
}

// NewIdentifier returns a new IdentifierNode with the given identifier name.
func newIdentifierNode(ident string) *IdentifierNode {
	return &IdentifierNode{NodeType: NodeIdentifier, Ident: ident}
}

// SetPos sets the position. NewIdentifier is a public method so we can't modify its signature.
// Chained for convenience.
// TODO: fix one day?
func (i *IdentifierNode) SetPos(pos Pos) *IdentifierNode {
	i.Pos = pos
	return i
}

func (i *IdentifierNode) String() string {
	return i.Ident
}

func (i *IdentifierNode) Copy() Node {
	return newIdentifierNode(i.Ident).SetPos(i.Pos)
}

// DotNode holds the special identifier '.'.
type DotNode struct {
	Pos
}

func newDotNode(pos Pos) *DotNode {
	return &DotNode{Pos: pos}
}

func (d *DotNode) Type() NodeType {
	return NodeDot
}

func (d *DotNode) String() string {
	return "."
}

func (d *DotNode) Copy() Node {
	return newDotNode(d.Pos)
}

// NilNode holds the special identifier 'nil' representing an untyped nil constant.
type NilNode struct {
	Pos
}

func newNilNode(pos Pos) *NilNode {
	return &NilNode{Pos: pos}
}

func (n *NilNode) Type() NodeType {
	return NodeNil
}

func (n *NilNode) String() string {
	return "nil"
}

func (n *NilNode) Copy() Node {
	return newNilNode(n.Pos)
}

// BoolNode holds a boolean constant.
type BoolNode struct {
	NodeType
	Pos
	True bool // The value of the boolean constant.
}

func newBoolNode(pos Pos, true bool) *BoolNode {
	return &BoolNode{NodeType: NodeBool, Pos: pos, True: true}
}

func (b *BoolNode) String() string {
	if b.True {
		return "true"
	}
	return "false"
}

func (b *BoolNode) Copy() Node {
	return newBoolNode(b.Pos, b.True)
}

// NumberNode holds a number: signed or unsigned integer, float, or complex.
// The value is parsed and stored under all the types that can represent the value.
// This simulates in a small amount of code the behavior of Go's ideal constants.
type NumberNode struct {
	NodeType
	Pos
	IsInt      bool       // Number has an integral value.
	IsUint     bool       // Number has an unsigned integral value.
	IsFloat    bool       // Number has a floating-point value.
	IsComplex  bool       // Number is complex.
	Int64      int64      // The signed integer value.
	Uint64     uint64     // The unsigned integer value.
	Float64    float64    // The floating-point value.
	Complex128 complex128 // The complex value.
	Text       string     // The original textual representation from the input.
}

func newNumberNode(pos Pos, text string, typ itemType) (*NumberNode, error) {
	n := &NumberNode{NodeType: NodeNumber, Pos: pos, Text: text}
	switch typ {
	case itemCharConstant:
		rune, _, tail, err := strconv.UnquoteChar(text[1:], text[0])
		if err != nil {
			return nil, err
		}
		if tail != "'" {
			return nil, fmt.Errorf("malformed character constant: %s", text)
		}
		n.Int64 = int64(rune)
		n.IsInt = true
		n.Uint64 = uint64(rune)
		n.IsUint = true
		n.Float64 = float64(rune) // odd but those are the rules.
		n.IsFloat = true
		return n, nil
		//	case itemComplex:
		//		// fmt.Sscan can parse the pair, so let it do the work.
		//		if _, err := fmt.Sscan(text, &n.Complex128); err != nil {
		//			return nil, err
		//		}
		//		n.IsComplex = true
		//		n.simplifyComplex()
		//		return n, nil
	}
	// Imaginary constants can only be complex unless they are zero.
	if len(text) > 0 && text[len(text)-1] == 'i' {
		f, err := strconv.ParseFloat(text[:len(text)-1], 64)
		if err == nil {
			n.IsComplex = true
			n.Complex128 = complex(0, f)
			n.simplifyComplex()
			return n, nil
		}
	}
	// Do integer test first so we get 0x123 etc.
	u, err := strconv.ParseUint(text, 0, 64) // will fail for -0; fixed below.
	if err == nil {
		n.IsUint = true
		n.Uint64 = u
	}
	i, err := strconv.ParseInt(text, 0, 64)
	if err == nil {
		n.IsInt = true
		n.Int64 = i
		if i == 0 {
			n.IsUint = true // in case of -0.
			n.Uint64 = u
		}
	}
	// If an integer extraction succeeded, promote the float.
	if n.IsInt {
		n.IsFloat = true
		n.Float64 = float64(n.Int64)
	} else if n.IsUint {
		n.IsFloat = true
		n.Float64 = float64(n.Uint64)
	} else {
		f, err := strconv.ParseFloat(text, 64)
		if err == nil {
			n.IsFloat = true
			n.Float64 = f
			// If a floating-point extraction succeeded, extract the int if needed.
			if !n.IsInt && float64(int64(f)) == f {
				n.IsInt = true
				n.Int64 = int64(f)
			}
			if !n.IsUint && float64(uint64(f)) == f {
				n.IsUint = true
				n.Uint64 = uint64(f)
			}
		}
	}
	if !n.IsInt && !n.IsUint && !n.IsFloat {
		return nil, fmt.Errorf("illegal number syntax: %q", text)
	}
	return n, nil
}

// simplifyComplex pulls out any other types that are represented by the complex number.
// These all require that the imaginary part be zero.
func (n *NumberNode) simplifyComplex() {
	n.IsFloat = imag(n.Complex128) == 0
	if n.IsFloat {
		n.Float64 = real(n.Complex128)
		n.IsInt = float64(int64(n.Float64)) == n.Float64
		if n.IsInt {
			n.Int64 = int64(n.Float64)
		}
		n.IsUint = float64(uint64(n.Float64)) == n.Float64
		if n.IsUint {
			n.Uint64 = uint64(n.Float64)
		}
	}
}

func (n *NumberNode) String() string {
	return n.Text
}

func (n *NumberNode) Copy() Node {
	nn := new(NumberNode)
	*nn = *n // Easy, fast, correct.
	return nn
}

// StringNode holds a string constant. The value has been "unquoted".
type StringNode struct {
	NodeType
	Pos
	Quoted string // The original text of the string, with quotes.
	Text   string // The string, after quote processing.
}

func newStringNode(pos Pos, orig, text string) *StringNode {
	return &StringNode{NodeType: NodeString, Pos: pos, Quoted: orig, Text: text}
}

func (s *StringNode) String() string {
	return s.Quoted
}

func (s *StringNode) Copy() Node {
	return newStringNode(s.Pos, s.Quoted, s.Text)
}

// endNode represents an {{end}} action.
// It does not appear in the final parse tree.
type endNode struct {
	Pos
}

func newEndNode(pos Pos) *endNode {
	return &endNode{Pos: pos}
}

func (e *endNode) Type() NodeType {
	return nodeEnd
}

func (e *endNode) String() string {
	return "{{end}}"
}

func (e *endNode) Copy() Node {
	return newEndNode(e.Pos)
}

// elseNode represents an {{else}} action. Does not appear in the final tree.
type elseNode struct {
	NodeType
	Pos
	Line int // The line number in the input (deprecated; kept for compatibility)
}

func newElseNode(pos Pos, line int) *elseNode {
	return &elseNode{NodeType: nodeElse, Pos: pos, Line: line}
}

func (e *elseNode) Type() NodeType {
	return nodeElse
}

func (e *elseNode) String() string {
	return "{{else}}"
}

func (e *elseNode) Copy() Node {
	return newElseNode(e.Pos, e.Line)
}

// BranchNode is the common representation of if.
type BranchNode struct {
	NodeType
	Pos
	Line int // The line number in the input (deprecated; kept for compatibility)
	//@@	Pipe     *PipeNode // The pipeline to be evaluated.
	List     *ListNode // What to execute if the value is non-empty.
	ElseList *ListNode // What to execute if the value is empty (nil if absent).
}

func (b *BranchNode) String() string {
	name := ""
	switch b.NodeType {
	case NodeIf:
		name = "if"
	default:
		panic("unknown branch type")
	}
	if b.ElseList != nil {
		return fmt.Sprintf("{{%s}}%s{{else}}%s{{end}}", name, b.List, b.ElseList)
	}
	return fmt.Sprintf("{{%s}}%s{{end}}", name, b.List)
}

// IfNode represents an {{if}} action and its commands.
type IfNode struct {
	BranchNode
}

func newIfNode(pos Pos, line int, list, elseList *ListNode) *IfNode {
	return &IfNode{BranchNode{NodeType: NodeIf, Pos: pos, Line: line, List: list, ElseList: elseList}}
}

func (i *IfNode) Copy() Node {
	return newIfNode(i.Pos, i.Line, i.List.CopyList(), i.ElseList.CopyList())
}
