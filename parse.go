// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parse builds parse trees for templates as defined by text/template
// and html/template. Clients should use those packages to construct templates
// rather than this one, which provides shared internal data structures not
// intended for general use.
package main

import (
	//@@	"bytes"
	"fmt"
	"log"
	"runtime"
	//	"strconv"
	"strings"
)

// Tree is the representation of a single parsed template.
type Tree struct {
	name string    // name of the tozzy processes and PP/PF represented by the tree.
	root *ListNode // top-level root of the tree.
	text string    // input texts
	// Parsing only; cleared after parse.
	funcs     []map[string]interface{}
	lex       *lexer
	token     [3]item // three-token lookahead for parser.
	peekCount int
}

// Parse returns a map from name to parse.Tree. If an error is encountered,
// parsing stops and an empty map is returned with the error.
func Parse(name, text, leftDelim, rightDelim string, funcs ...map[string]interface{}) (treeSet map[string]*Tree, err error) {
	treeSet = make(map[string]*Tree)
	t := NewTree(name)
	t.text = text

	/*
	 * Problem!!: deadlock!!
	 */
	_, err = t.Parse(text, leftDelim, rightDelim, treeSet, funcs...)	// wholefile, "%%", "%%", treeset, builtins
	
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		log.Fatal("PARSE ERROR", file, " ", line+1)
	}

	return
}

// New allocates a new parse tree with the given name.
func NewTree(name string, funcs ...map[string]interface{}) *Tree {
	return &Tree{
		name:  name,
		funcs: funcs,
	}
}

// IsEmptyTree reports whether this tree (node) is empty of everything but space.
func IsEmptyTree(n Node) bool {
	switch n := n.(type) {
	case nil:
		return true
	case *IfNode:
	case *ListNode:
		for _, node := range n.Nodes {
			if !IsEmptyTree(node) {
				return false
			}
		}
		return true
		//@@	case *TextNode:
		//		return len(bytes.TrimSpace(n.Text)) == 0
	default:
		panic("unknown node: " + n.String())
	}
	return false
}

// Parse parses the template definition string to construct a representation of
// the template for execution. If either action delimiter string is empty, the
// default ("{{" or "}}") is used. Embedded template definitions are added to
// the treeSet map.
func (t *Tree) Parse(text, leftDelim, rightDelim string, treeSet map[string]*Tree, funcs ...map[string]interface{}) (tree *Tree, err error) {
	defer t.recover(&err)
	t.startParse(funcs, lex(t.name, text, leftDelim, rightDelim))
	t.text = text

	/*
	 * problem here!: Deadlock!!
	 */
	t.parse(treeSet)
	
	t.add(treeSet)
	t.stopParse()
	return t, nil
}

func (t *Tree) lexerTest() {
	for t.peek().typ != itemEOF {
		if t.peek().typ == itemLeftDelim {

			item := t.nextNonSpace()
			for item.typ != itemEOF {
				fmt.Println("================== --->", item.typ, "<---", item.val)
				
				// =============================================================================
				/*
				if item.typ == itemError {
					fmt.Println(">>> item.typ: itemError <<<")
					break
				}
				*/
				// =============================================================================
						
				if item.typ == itemRightDelim {
					break
				}
				item = t.nextNonSpace()
			}
			log.Fatal("DONE FOR NOW")
		}
	}
}

// parse is the top-level parser for a input
// next() and peek() gets new item from lexer
// It runs to EOF.
func (t *Tree) parse(treeSet map[string]*Tree) {
	t.root = newListNode(t.peek().pos)
	
	/*
	 * Here DEADLOCK!!!
	 */
	t.lexerTest() //@@ TODO -JUST TO TEST LEXER

	for t.peek().typ != itemEOF {
		if t.peek().typ == itemLeftDelim {
			delim := t.next() // get next token from lexer via channel "<-items"

			if t.nextNonSpace().typ == itemIdentifier {
				newT := NewTree("procDef") // name will be updated once we know it.
				newT.text = t.text
				print("DDDDDDDDDDDD", t.text[t.lex.pos:], "SSSSSSSSSS")
				newT.startParse(t.funcs, t.lex)
				newT.parseProcDef(treeSet)
				continue
			}
			t.backup2(delim)
		}
		n := t.itemNode() // tree node
		if n.Type() == nodeEnd {
			t.errorf("unexpected %s", n)
		}
		t.root.append(n)
	}
}

//@@func (t *Tree) parse(treeSet map[string]*Tree) (next Node) {
//	t.root = newList(t.peek().pos)
//	for t.peek().typ != itemEOF {
//		if t.peek().typ == itemLeftDelim {
//			delim := t.next()
//			if t.nextNonSpace().typ == itemDefine {
//				newT := NewTree("definition") // name will be updated once we know it.
//				newT.text = t.text
//				newT.startParse(t.funcs, t.lex)
//				newT.parseDefinition(treeSet)
//				continue
//			}
//			t.backup2(delim)
//		}
//		n := t.texts()
//		if n.Type() == nodeEnd {
//			t.errorf("unexpected %s", n)
//		}
//		t.root.append(n)
//	}
//	return nil
//}

// startParse initializes the parser, using the lexer.
func (t *Tree) startParse(funcs []map[string]interface{}, lex *lexer) {
	t.root = nil
	t.lex = lex
	t.funcs = funcs
}

// stopParse terminates parsing.
func (t *Tree) stopParse() {
	t.lex = nil
	t.funcs = nil
}

//	token
func (t *Tree) itemNode() Node {
	fmt.Println("@@ (t *Tree) itemNode")
	switch token := t.nextNonSpace(); token.typ {
	case itemBool: // boolean constant
	case itemChar: // printable ASCII character; grab bag for comma etc.
	case itemCharConstant: // character constant
	case itemEOF:
	case itemIdentifier: // alphanumeric identifier not starting with '.'
		return newIdentifierNode(token.val)
	case itemLeftDelim: // left action delimiter
	case itemLeftParen: // '(' inside action
	case itemNumber: // simple number, including imaginary
	case itemRawString: // raw quoted string (includes quotes)
	case itemRightDelim: // right action delimiter
	case itemRightParen: // ')' inside action
	case itemSpace: // run of spaces separating arguments
	case itemString: // quoted string (includes quotes)
	case itemVariable: // variable starting with '$', such as '$' or  '$1' or '$hello'

	// Keywords appear after all the rest.
	case itemKeyword: // used only to delimit the keywords
	case itemColon: // ':'
	case itemColonEquals: // colon-equals (':=') introducing a declaration
	case itemDot: // the cursor, spelled '.'
	case itemElse: // else keyword
	case itemEnd: // end keyword
	case itemEquals: // '='
	case itemIf: // if keyword
	case itemNil: // the untyped nil constant, easiest to treat as a keyword
	default:
		t.unexpected(token, "input")
	}
	return nil
}

func (t *Tree) parseProcDef(treeSet map[string]*Tree) {
	fmt.Println("@@ (t *Tree) parseProcDef")

	// TODO

	t.add(treeSet)
	t.stopParse()

	//	const context = "define clause"
	//	name := t.expectOneOf(itemString, itemRawString, context)
	//	var err error
	//	t.name, err = strconv.Unquote(name.val)
	//	if err != nil {
	//		t.error(err)
	//	}
	//	t.expect(itemRightDelim, context)
	//	var end Node
	//	t.root, end = t.itemList()
	//	if end.Type() != nodeEnd {
	//		t.errorf("unexpected %s in %s", end, context)
	//	}
	//	t.add(treeSet)
	//	t.stopParse()
}

// add adds tree to the treeSet.
func (t *Tree) add(treeSet map[string]*Tree) {
	fmt.Println("@@ (t *Tree) add")
	tree := treeSet[t.name]
	if tree == nil || IsEmptyTree(tree.root) {
		treeSet[t.name] = t
		return
	}
	if !IsEmptyTree(t.root) {
		t.errorf("tozzy: multiple definition of process %q", t.name)
	}
}

//// Copy returns a copy of the Tree. Any parsing state is discarded.
//@@func (t *Tree) Copy() *Tree {
//	fmt.Println("@@ (t *Tree) Copy")
//	if t == nil {
//		return nil
//	}
//	return &Tree{
//		name: t.name,
//		root: t.root.CopyList(),
//		text: t.text,
//	}
//}

// next returns the next token.
func (t *Tree) next() item {
	fmt.Println("NEXT() +++++++++++++++++++++++++++++++++++++++++++++")
		
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		fmt.Println("LEX. NEXTITEM() right after this. ==========================================")
		t.token[0] = t.lex.nextItem()
	}
	fmt.Println("@@ (t *Tree) next", "(", t.token[t.peekCount], ")")

	return t.token[t.peekCount]
}

// peek returns but does not consume the next token.
func (t *Tree) peek() item {
	if t.peekCount > 0 {
		return t.token[t.peekCount-1]
	}
	t.peekCount = 1
	t.token[0] = t.lex.nextItem() // get nextItem from lexer via channel "<-items"
	return t.token[0]
}

// backup backs the input stream up one token.
func (t *Tree) backup() {
	t.peekCount++
}

// backup2 backs the input stream up two tokens.
// The zeroth token is already there.
func (t *Tree) backup2(t1 item) {
	t.token[1] = t1
	t.peekCount = 2
}

// backup3 backs the input stream up three tokens
// The zeroth token is already there.
func (t *Tree) backup3(t2, t1 item) { // Reverse order: we're pushing back.
	t.token[1] = t1
	t.token[2] = t2
	t.peekCount = 3
}

// nextNonSpace returns the next non-space token.
func (t *Tree) nextNonSpace() (token item) {
	for {
		fmt.Println("UUUUUUUUUUUUUUUUUUUUUUUUUUUUUU")
		token = t.next()
		if token.typ != itemSpace {
			break
		}
	}
	return token
}

// peekNonSpace returns but does not consume the next non-space token.
func (t *Tree) peekNonSpace() (token item) {
	for {
		token = t.next()
		if token.typ != itemSpace {
			break
		}
	}
	t.backup()
	return token
}

//======================================================================
// Parsing.
//======================================================================

// ErrorContext returns a textual representation of the location of the node in the input text.
func (t *Tree) ErrorContext(n Node) (location, context string) {
	fmt.Println("@@ (t *Tree) Error")
	pos := int(n.Position())
	text := t.text[:pos]
	byteNum := strings.LastIndex(text, "\n")
	if byteNum == -1 {
		byteNum = pos // On first line.
	} else {
		byteNum++ // After the newline.
		byteNum = pos - byteNum
	}
	lineNum := 1 + strings.Count(text, "\n")
	context = n.String()
	if len(context) > 20 {
		context = fmt.Sprintf("%.20s...", context)
	}
	return fmt.Sprintf("%s:%d:%d", lineNum, byteNum), context
}

// errorf formats the error and terminates processing.
func (t *Tree) errorf(format string, args ...interface{}) {
	fmt.Println("@@ (t *Tree) errorf")
	t.root = nil
	format = fmt.Sprintf("template: %s:%d: %s", t.lex.lineNumber(), format)
	panic(fmt.Errorf(format, args...))
}

// error terminates processing.
func (t *Tree) error(err error) {
	fmt.Println("@@ (t *Tree) error")
	t.errorf("%s", err)
}

// expect consumes the next token and guarantees it has the required type.
func (t *Tree) expect(expected itemType, context string) item {
	fmt.Println("@@ (t *Tree) expect")
	token := t.nextNonSpace()
	if token.typ != expected {
		t.unexpected(token, context)
	}
	return token
}

// expectOneOf consumes the next token and guarantees it has one of the required types.
func (t *Tree) expectOneOf(expected1, expected2 itemType, context string) item {
	fmt.Println("@@ (t *Tree) expectOneOf")
	token := t.nextNonSpace()
	if token.typ != expected1 && token.typ != expected2 {
		t.unexpected(token, context)
	}
	return token
}

// unexpected complains about the token and terminates processing.
func (t *Tree) unexpected(token item, context string) {
	fmt.Println("@@ (t *Tree) unexpected")
	t.errorf("unexpected %s in %s", token, context)
}

// recover is the handler that turns panics into returns from the top level of Parse.
func (t *Tree) recover(errp *error) {
	fmt.Println("@@ (t *Tree) recover")
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		if t != nil {
			t.stopParse()
		}
		*errp = e.(error)
	}
	return
}

// itemList:
//	textOrAction*
// Terminates at {{end}} or {{else}}, returned separately.
func (t *Tree) itemList() (list *ListNode, next Node) {
	fmt.Println("@@ (t *Tree) itemList")
	list = newListNode(t.peekNonSpace().pos)
	for t.peekNonSpace().typ != itemEOF {
		n := t.itemNode()
		switch n.Type() {
		case nodeEnd, nodeElse:
			return list, n
		}
		list.append(n)
	}
	t.errorf("unexpected EOF")
	return
}

func (t *Tree) parseControl(allowElseIf bool, context string) (pos Pos, line int, list, elseList *ListNode) {
	fmt.Println("@@ (t *Tree) parseControl")
	//@@	defer t.popVars(len(t.vars))
	line = t.lex.lineNumber()
	var next Node
	list, next = t.itemList()
	switch next.Type() {
	case nodeEnd: //done
	case nodeElse:
		if allowElseIf {
			// Special case for "else if". If the "else" is followed immediately by an "if",
			// the elseControl will have left the "if" token pending. Treat
			//	{{if a}}_{{else if b}}_{{end}}
			// as
			//	{{if a}}_{{else}}{{if b}}_{{end}}{{end}}.
			// To do this, parse the if as usual and stop at it {{end}}; the subsequent{{end}}
			// is assumed. This technique works even for long if-else-if chains.
			// TODO: Should we allow else-if in with and range?
			if t.peek().typ == itemIf {
				t.next() // Consume the "if" token.
				elseList = newListNode(next.Position())
				elseList.append(t.ifControl())
				// Do not consume the next item - only one {{end}} required.
				break
			}
		}
		elseList, next = t.itemList()
		if next.Type() != nodeEnd {
			t.errorf("expected end; found %s", next)
		}
	}
	return next.Position(), line, list, elseList
}

// If:
//	{{if pipeline}} itemList {{end}}
//	{{if pipeline}} itemList {{else}} itemList {{end}}
// If keyword is past.
func (t *Tree) ifControl() Node {
	fmt.Println("@@ (t *Tree) ifControl")
	return newIfNode(t.parseControl(true, "if"))
}

// End:
//	{{end}}
// End keyword is past.
func (t *Tree) endControl() Node {
	fmt.Println("@@ (t *Tree) endControl")
	return newEndNode(t.expect(itemRightDelim, "end").pos)
}

// Else:
//	{{else}}
// Else keyword is past.
func (t *Tree) elseControl() Node {
	fmt.Println("@@ (t *Tree) elseControl")
	// Special case for "else if".
	peek := t.peekNonSpace()
	if peek.typ == itemIf {
		// We see "{{else if ... " but in effect rewrite it to {{else}}{{if ... ".
		return newElseNode(peek.pos, t.lex.lineNumber())
	}
	return newElseNode(t.expect(itemRightDelim, "else").pos, t.lex.lineNumber())
}

// hasFunction reports if a function name exists in the Tree's maps.
func (t *Tree) hasFunction(name string) bool {
	fmt.Println("@@ (t *Tree) hasFunction")
	for _, funcMap := range t.funcs {
		if funcMap == nil {
			continue
		}
		if funcMap[name] != nil {
			return true
		}
	}
	return false
}
