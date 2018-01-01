package eval

import (
	"errors"
	"unicode/utf8"

	"github.com/elves/elvish/eval/types"
	"github.com/elves/elvish/parse"
	"github.com/xiaq/persistent/hash"
)

// String is just a string.
type String string

var (
	_ types.Value    = String("")
	_ types.ListLike = String("")
)

var ErrReplacementMustBeString = errors.New("replacement must be string")

func (String) Kind() string {
	return "string"
}

func (s String) Repr(int) string {
	return parse.Quote(string(s))
}

func (s String) Equal(rhs interface{}) bool {
	return s == rhs
}

func (s String) Hash() uint32 {
	return hash.String(string(s))
}

func (s String) String() string {
	return string(s)
}

func (s String) Len() int {
	return len(string(s))
}

func (s String) IndexOne(idx types.Value) types.Value {
	i, j := s.index(idx)
	return s[i:j]
}

func (s String) Assoc(idx, v types.Value) types.Value {
	i, j := s.index(idx)
	repl, ok := v.(String)
	if !ok {
		throw(ErrReplacementMustBeString)
	}
	return s[:i] + repl + s[j:]
}

func (s String) index(idx types.Value) (int, int) {
	slice, i, j := types.ParseAndFixListIndex(types.ToString(idx), len(s))
	r, size := utf8.DecodeRuneInString(string(s[i:]))
	if r == utf8.RuneError {
		throw(types.ErrBadIndex)
	}
	if slice {
		if r, _ := utf8.DecodeLastRuneInString(string(s[:j])); r == utf8.RuneError {
			throw(types.ErrBadIndex)
		}
		return i, j
	}
	return i, i + size
}

func (s String) Iterate(f func(v types.Value) bool) {
	for _, r := range s {
		b := f(String(string(r)))
		if !b {
			break
		}
	}
}

func resolve(s string, ec *Frame) Fn {
	// NOTE: This needs to be kept in sync with the resolution algorithm used in
	// (*compiler).form.

	// Try variable
	explode, ns, name := ParseVariable(string(s))
	if !explode {
		if v := ec.ResolveVar(ns, name+FnSuffix); v != nil {
			if caller, ok := v.Get().(Fn); ok {
				return caller
			}
		}
	}

	// External command
	return ExternalCmd{string(s)}
}
