package htmltpl

import "strings"

type dataArgKind int

const (
	dataArgNil            dataArgKind = iota // nil, omitted, or "."
	dataArgDotPath                          // .Foo, .Foo.Bar, .Method (no args)
	dataArgMethodWithArgs                   // .Method "arg", .Foo.Method arg1 arg2
	dataArgLiteral                          // "hello", 42, .5, true, false
	dataArgOther                            // $var, pipe, func call
)

// classifyDataArg classifies the data argument of a {{template}} or {{block}} call.
func classifyDataArg(data string) dataArgKind {
	// No data, nil, or bare dot — same context
	if data == "" || data == "nil" || data == "." {
		return dataArgNil
	}
	// Starts with dot
	if data[0] == '.' {
		if !isDotRef(data) {
			return dataArgLiteral // .5, .42 (number literal)
		}
		// Has spaces → method call with args like ".Method arg1"
		if strings.Contains(data, " ") {
			return dataArgMethodWithArgs
		}
		return dataArgDotPath // .Foo, .Foo.Bar, .Method (no args)
	}
	// Quoted string
	if data[0] == '"' || data[0] == '\'' || data[0] == '`' {
		return dataArgLiteral
	}
	// Number (including negative)
	if isDigit(data[0]) || (data[0] == '-' && len(data) > 1 && isDigit(data[1])) {
		return dataArgLiteral
	}
	// Boolean
	if data == "true" || data == "false" {
		return dataArgLiteral
	}
	// Everything else: $var, pipe, func call
	return dataArgOther
}

// isDotRef reports whether s (starting with '.') is a dot reference,
// not a number literal like ".4".
// Bare "." → true. ".Foo" → true. ".4" → false.
func isDotRef(s string) bool {
	if len(s) == 1 {
		return true // bare "."
	}
	return !isDigit(s[1]) // second char is digit → number literal
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
