// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
)

/*
 * only for debugging
 */
var sendCount int
var receiveCount int

var errorSendCount int
var errorReceiveCount int


// item represents a token or text string returned from the scanner.
type item struct {
	typ itemType // The type of this item.
	pos Pos      // The starting position, in bytes, of this item in the input string.
	val string   // The value of this item.
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		fallthrough
	case i.typ == itemIdentifier:
		return i.val
	case i.typ > itemKeyword:
		return fmt.Sprintf("<%s>", i.val)
	case len(i.val) > 10:
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

// itemType identifies the type of lex items.
type itemType int

const (
	itemError itemType = iota // error occurred; value is text of error

	itemBang               // '!'
	itemBool               // boolean constant
	itemChar               // printable ASCII character; grab bag for comma etc.
	itemCharConstant       // character constant
	itemColon              // ':'
	itemColonEquals        // colon-equals (':=') introducing a declaration
	itemDot                // the cursor, spelled '.'
	itemEOF                // EOF
	itemEquals             // '='
	itemIdentifier         // alphanumeric identifier not starting with '.'
	itemLeftAngleBracket   // '<'
	itemLeftCurlyBracket   // '{'
	itemLeftDelim          // left action delimiter
	itemLeftParen          // '(' inside action
	itemLeftSquareBracket  // '['
	itemLogicAND           // '&'
	itemLogicOR            // '|'
	itemMinus              // '-'
	itemNumber             // simple number, including imaginary
	itemParallel           // '||'
	itemPlus               // '+' could be either non-deterministic choice or plus
	itemQuestionMark       // '?'
	itemRawString          // raw quoted string (includes quotes)
	itemRightAngleBracket  // '>'
	itemRightCurlyBracket  // '}'
	itemRightDelim         // right action delimiter
	itemRightParen         // ')' inside action
	itemRightSquareBracket // ']'
	itemSpace              // run of spaces separating arguments
	itemString             // quoted string (includes quotes)
	itemVariable           // variable starting with '$', such as '$' or  '$1' or '$hello'

	// Keywords appear after all the rest. (i.e. reserved words)
	itemKeyword // used only to delimit the keywords
	itemElse    // else keyword
	itemEnd     // end keyword
	itemIf      // if keyword
	itemNil     // the untyped nil constant, easiest to treat as a keyword
)

var key = map[string]itemType{
	"else": itemElse,
	"end":  itemEnd,
	"if":   itemIf,
	"nil":  itemNil,
}

const eof = -1

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name       string    // the name of the input; used only for error reports
	input      string    // the string being scanned
	leftDelim  string    // start of action
	rightDelim string    // end of action
	state      stateFn   // the next lexing function to enter
	pos        Pos       // current position in the input
	start      Pos       // start position of this item
	width      Pos       // width of last rune read from input
	lastPos    Pos       // position of most recent item returned by nextItem
	items      chan item // synchronous channel of scanned items
	parenDepth int       // nesting depth of ( ) exprs
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	//@@ fmt.Println("@@ (l *lexer) next", "[", string(r), "]")
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	//@@ fmt.Println("@@ (l *lexer) peek", "[", string(r), "]")
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

//----------------------------------------------------------
// emit passes an item back to the client.
// send item to channel "items"
func (l *lexer) emit(t itemType) {
	//@@ fmt.Println("@@ (l *lexer) emit", t)
	
	
	// ==========================================================================================
	
	sendItem := item{t, l.start, l.input[l.start:l.pos]}
	fmt.Println("sendItem:", sendItem)
	l.items <- sendItem
	//l.items <- item{t, l.start, l.input[l.start:l.pos]}
	
			
	// ==========================================================================================
			
	sendCount++
	fmt.Println("send:", sendCount)
	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// lineNumber reports which line we're on, based on the position of
// the previous item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (l *lexer) lineNumber() int {
	//@@ fmt.Println("@@ (l *lexer) lineNumber")
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	//@@ fmt.Println("@@ (l *lexer) errorf")
	
	// ================================================================================================

	errorItem := item{itemError, l.start, fmt.Sprintf(format, args...)}
	fmt.Println("errorItem:", errorItem)
	fmt.Println("errorItem Type:", errorItem.typ)
	l.items <- errorItem
	//l.items <- item{itemError, l.start, fmt.Sprintf(format, args...)}
	
	
	// ================================================================================================
	
	errorSendCount++
	fmt.Println("errorSend:", errorSendCount)

	close(l.items)

	return nil
}

//------------------------------------------------------
// nextItem returns the next item from the input.
// next and peek in parse.go calls this function to get nextItem
func (l *lexer) nextItem() item {
	if _, file, _, _ := runtime.Caller(1); strings.Index(file, "parse.go") == -1 {
		log.Fatal("\"nextItem\" should only called by \"next\" and \"peek\" in parse.go")
	}

	item := <-l.items

	fmt.Println("receiveItem:", item)
	receiveCount++
	fmt.Println("receive:", receiveCount)
	
	l.lastPos = item.pos
	//@@ fmt.Println("@@ (l *lexer) nextItem", "(", item.val, ")")
	return item
}

// atTerminator reports whether the input is at valid termination character to
// appear after an identifier. Breaks .X.Y into two pieces. Also catches cases
// like "$x+2" not being acceptable without a space, in case we decide one
// day to implement arithmetic.
func (l *lexer) atTerminator() bool {
	//@@ fmt.Println("@@ atTerminator")
	r := l.peek()
	if isSpace(r) || isEndOfLine(r) {
		return true
	}
	switch r {
	case eof, '.', ',', '|', ':', ')', '(', '!', '?', '+', '-',
		'{', '}', '[', ']', '<', '>':
		return true
	}
	// Does r start the delimiter? This can be ambiguous (with delim=="//", $x/2 will
	// succeed but should fail) but only in extremely rare cases caused by willfully
	// bad choice of delimiter.
	if rd, _ := utf8.DecodeRuneInString(l.rightDelim); rd == r {
		return true
	}
	return false
}

func (l *lexer) scanNumber() bool {
	//@@ fmt.Println("@@ (l *lexer) scanNumber")
	// Optional leading sign.
	l.accept("+-")
	// Is it hex?
	digits := "0123456789"
	if l.accept("0") && l.accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}
	// Is it imaginary?
	l.accept("i")
	// Next thing mustn't be alphanumeric.
	if isAlphaNumeric(l.peek()) {
		_ = l.next()
		return false
	}
	return true
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	//@@ fmt.Println("@@ (l *lexer) run")
	for l.state = lexStart; l.state != nil; {
		l.state = l.state(l)
	}
}

// lex creates a new scanner for the input string.
func lex(name, input, left, right string) *lexer {
	//@@ fmt.Println("@@ lex")
	if left == "" {
		left = leftDelim
	}
	if right == "" {
		right = rightDelim
	}
	l := &lexer{
		name:       name,
		input:      input,
		leftDelim:  left,
		rightDelim: right,
		items:      make(chan item, 0),
	}
	go l.run()
	return l
}

//========================================================
// state functions
//========================================================
const (
	leftDelim    = "%%"
	rightDelim   = "%%"
	leftComment  = "/*"
	rightComment = "*/"
)

// lexStart scans until an opening action delimiter, "%%".
func lexStart(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexStart")
	for {
		if strings.HasPrefix(l.input[l.pos:], l.leftDelim) {
			if l.pos > l.start {
				//@@				l.emit(itemText)
				// do nothing, in fact this should be assert(false)
			}
			return lexLeftDelim
		}
		if l.next() == eof {
			break
		}
	}
	// Correctly reached EOF.
	if l.pos > l.start {
		//@@		l.emit(itemText)
		// do nothing, in fact this should be assert(false)
	}
	l.emit(itemEOF)
	return nil
}

// lexLeftDelim scans the left delimiter, which is known to be present.
func lexLeftDelim(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexLeftDelim")
	l.pos += Pos(len(l.leftDelim))
	//	if strings.HasPrefix(l.input[l.pos:], leftComment) {
	//		return lexComment
	//	}
	l.emit(itemLeftDelim) // notify to parser that "left delimeter" is found
	//l.parenDepth = 0
	return lexMisc
}

func lexMisc(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexMisc")
	// Either number, quoted string, or identifier.
	// Spaces separate arguments; runs of spaces turn into itemSpace.
	if strings.HasPrefix(l.input[l.pos:], l.rightDelim) {
		if l.parenDepth == 0 {
			return lexRightDelim
		}
		return l.errorf("unclosed left paren")
	}
	switch r := l.next(); {
	case r == eof:
		return l.errorf("unclosed action")
	case isEndOfLine(r):
		return lexSpace
	case isSpace(r):
		return lexSpace
	case r == '!':
		l.emit(itemBang)
	case r == '?':
		l.emit(itemQuestionMark)
	case r == '+':
		l.emit(itemPlus)
	case r == '-':
		l.emit(itemMinus)
	case r == ':':
		l.emit(itemColon)
		//		if l.next() != '=' {
		//			return l.errorf("expected :=")
		//		}
		//		l.emit(itemColonEquals)
	case r == '&':
		l.emit(itemLogicAND)
	case r == '|':
		if l.next() == '|' {
			l.emit(itemParallel)
		} else {
			l.emit(itemLogicOR)
		}
	case r == '{':
		l.emit(itemLeftCurlyBracket)
	case r == '}':
		l.emit(itemRightCurlyBracket)
	case r == '"':
		return lexQuote
	case r == '`':
		return lexRawQuote
		//	case r == '$':
		//		return lexVariable
	case r == '\'':
		return lexChar
		//	case r == '.':
		//		// special look-ahead for ".field" so we don't break l.backup().
		//		if l.pos < Pos(len(l.input)) {
		//			r := l.input[l.pos]
		//			if r < '0' || '9' < r {
		//				return lexField
		//			}
		//		}
		//		fallthrough // '.' can start a number.
	case r == '+' || r == '-' || ('0' <= r && r <= '9'):
		l.backup()
		return lexNumber
	case isAlphaNumeric(r):
		l.backup()
		return lexIdentifier
	case r == '(':
		l.emit(itemLeftParen)
		l.parenDepth++
	case r == ')':
		l.emit(itemRightParen)
		l.parenDepth--
		if l.parenDepth < 0 {
			return l.errorf("unexpected right paren %#U", r)
		}
	case r <= unicode.MaxASCII && unicode.IsPrint(r):
		l.emit(itemChar)
	default:
		return l.errorf("unrecognized character in action: %#U", r)
	}
	return lexMisc
}

// lexComment scans a comment. The left comment marker is known to be present.
func lexComment(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexComment")
	l.pos += Pos(len(leftComment))
	i := strings.Index(l.input[l.pos:], rightComment)
	if i < 0 {
		return l.errorf("unclosed comment")
	}
	l.pos += Pos(i + len(rightComment))
	if !strings.HasPrefix(l.input[l.pos:], l.rightDelim) {
		return l.errorf("comment ends before closing delimiter")

	}
	l.pos += Pos(len(l.rightDelim))
	l.ignore()
	return lexMisc
}

// lexRightDelim scans the right delimiter, which is known to be present.
func lexRightDelim(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexRightDelim")
	l.pos += Pos(len(l.rightDelim))
	l.emit(itemRightDelim)
	return lexMisc
}

// lexSpace scans a run of space characters.
// One space has already been seen.
func lexSpace(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexSpace")
	for isSpace(l.peek()) {
		l.next()
	}
	l.emit(itemSpace)
	return lexMisc
}

// lexIdentifier scans an alphanumeric.
func lexIdentifier(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexIdentifier")
Loop:
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb.
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			if !l.atTerminator() {
				return l.errorf("bad character %#U", r)
			}
			switch {
			case key[word] > itemKeyword:
				l.emit(key[word])
			case word == "true", word == "false":
				l.emit(itemBool)
			default:
				l.emit(itemIdentifier)
			}
			break Loop
		}
	}
	return lexMisc
}

// lexChar scans a character constant. The initial quote is already
// scanned. Syntax checking is done by the parser.
func lexChar(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexChar")
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated character constant")
		case '\'':
			break Loop
		}
	}
	l.emit(itemCharConstant)
	return lexMisc
}

// lexNumber scans a number: decimal, octal, hex, float, or imaginary. This
// isn't a perfect number scanner - for instance it accepts "." and "0x0.2"
// and "089" - but when it's wrong the input is invalid and the parser (via
// strconv) will notice.
func lexNumber(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexNumber")
	if !l.scanNumber() {
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	}
	l.emit(itemNumber)
	return lexMisc
}

// lexQuote scans a quoted string.
func lexQuote(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexQuote")
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break // break out switch
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated quoted string")
		case '"':
			break Loop // break out for
		}
	}
	l.emit(itemString)
	return lexMisc
}

// lexRawQuote scans a raw quoted string.
func lexRawQuote(l *lexer) stateFn {
	//@@ fmt.Println("@@ lexRawQuote")
Loop:
	for {
		switch l.next() {
		case eof, '\n':
			return l.errorf("unterminated raw quoted string")
		case '`':
			break Loop
		}
	}
	l.emit(itemRawString)
	return lexMisc
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	//@@ fmt.Println("@@ isSpace")
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	//@@ fmt.Println("@@ isEndOfLine")
	return r == '\r' || r == '\n'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	//@@ fmt.Println("@@ isAlphaNumeric (", string(r), ")")
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
