package yamlpath

import (
	"strings"
)

// NormalizeRootPath normalizes the root path of the YAML document
// by removing the first array marker (if it exists).
func NormalizeRootPath(path string) string {
	if len(path) < 2 {
		return path
	}
	if path[0] == '$' && path[1] == '[' {
		closingBracketIndex := strings.IndexRune(path, ']')
		if closingBracketIndex != -1 {
			return "$" + path[closingBracketIndex+1:]
		}
	}
	return path
}

func NormalizePath(path string) string {
	if len(path) == 0 {
		return path
	}
	var arrStart bool
	normalized := make([]byte, 0, len(path))
	for i := range path {
		switch {
		case path[i] == '[':
			arrStart = true
		case path[i] == ']':
			normalized = append(normalized, '*')
			arrStart = false
		case arrStart:
			continue // skip array index
		}
		normalized = append(normalized, path[i])
	}
	return string(normalized)
}
