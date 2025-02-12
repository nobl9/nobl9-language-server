package yamlpath

import (
	"strconv"

	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
)

// FromString creates a [Path] from string.
//
// Only a subset of rules are supported in comparison to JSONPath or YAMLPath:
// $     		: the root object/element
// .<child>     : child operator
// [num] 		: indexed element of an array
//
// If you want to use reserved characters such as `.` as a key name,
// enclose them in single quotation as follows ( $.foo.'bar.baz'.name ).
// If you want to use a single quote with reserved characters, escape it with `\` ( $.foo.'bar.baz\'s value'.name ).
func FromString(s string) (*Path, error) {
	buf := []rune(s)
	length := len(buf)
	cursor := 0
	builder := &PathBuilder{path: s}
	for cursor < length {
		switch buf[cursor] {
		case '$':
			builder = builder.Root()
			cursor++
		case '.':
			b, buf, c, err := parsePathDot(builder, buf, cursor)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse path of dot")
			}
			length = len(buf)
			builder = b
			cursor = c
		case '[':
			b, buf, c, err := parsePathIndex(builder, buf, cursor)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse path of index")
			}
			length = len(buf)
			builder = b
			cursor = c
		default:
			return nil, errors.Wrapf(yaml.ErrInvalidPathString, "invalid path at %d", cursor)
		}
	}
	return builder.Build(), nil
}

func parsePathDot(b *PathBuilder, buf []rune, cursor int) (*PathBuilder, []rune, int, error) {
	length := len(buf)
	cursor++ // skip . character
	start := cursor

	// if started single quote, looking for end single quote char
	if cursor < length && buf[cursor] == '\'' {
		return parseQuotedKey(b, buf, cursor)
	}
	for ; cursor < length; cursor++ {
		c := buf[cursor]
		switch c {
		case '$':
			return nil, nil, 0, errors.Wrap(yaml.ErrInvalidPathString, "specified '$' after '.' character")
		case '.', '[':
			goto end
		case ']':
			return nil, nil, 0, errors.Wrap(yaml.ErrInvalidPathString, "specified ']' after '.' character")
		}
	}
end:
	if start == cursor {
		return nil, nil, 0, errors.Wrap(yaml.ErrInvalidPathString, "cloud not find by empty key")
	}
	return b.Child(string(buf[start:cursor])), buf, cursor, nil
}

func parseQuotedKey(b *PathBuilder, buf []rune, cursor int) (*PathBuilder, []rune, int, error) {
	cursor++ // skip single quote
	start := cursor
	length := len(buf)
	var foundEndDelim bool
	for ; cursor < length; cursor++ {
		switch buf[cursor] {
		case '\\':
			buf = append(append([]rune{}, buf[:cursor]...), buf[cursor+1:]...)
			length = len(buf)
		case '\'':
			foundEndDelim = true
			goto end
		}
	}
end:
	if !foundEndDelim {
		return nil, nil, 0, errors.Wrap(yaml.ErrInvalidPathString, "could not find end delimiter for key")
	}
	if start == cursor {
		return nil, nil, 0, errors.Wrap(yaml.ErrInvalidPathString, "could not find by empty key")
	}
	selector := buf[start:cursor]
	cursor++
	if cursor < length {
		switch buf[cursor] {
		case '$':
			return nil, nil, 0, errors.Wrap(yaml.ErrInvalidPathString, "specified '$' after '.' character")
		case ']':
			return nil, nil, 0, errors.Wrap(yaml.ErrInvalidPathString, "specified ']' after '.' character")
		}
	}
	return b.Child(string(selector)), buf, cursor, nil
}

func parsePathIndex(b *PathBuilder, buf []rune, cursor int) (*PathBuilder, []rune, int, error) {
	length := len(buf)
	cursor++ // skip '[' character
	if length <= cursor {
		return nil, nil, 0, errors.Wrap(yaml.ErrInvalidPathString, "unexpected end of YAML Path")
	}
	c := buf[cursor]
	switch c {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		start := cursor
		cursor++
		for ; cursor < length; cursor++ {
			switch buf[cursor] {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				continue
			}
			break
		}
		if buf[cursor] != ']' {
			return nil, nil, 0, errors.Wrapf(
				yaml.ErrInvalidPathString,
				"invalid character %s at %d",
				string(buf[cursor]), cursor,
			)
		}
		numStr := string(buf[start:cursor])
		num, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return nil, nil, 0, errors.Wrapf(err, "failed to parse number")
		}
		// #nosec G115
		return b.Index(uint(num)), buf, cursor + 1, nil
	}
	return nil, nil, 0, errors.Wrapf(yaml.ErrInvalidPathString, "invalid character %s at %d", string(c), cursor)
}
