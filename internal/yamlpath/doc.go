// Package yamlpath was copied, refactored and trimmed from:
// https://github.com/goccy/go-yaml/blob/4653a1bb5c0047bb37280ac341e2f091cb44352f/path.go
//
// The main difference with the original is that instead of returning
// a child node value, it returns the node itself.
// For example:
//
//	# Given YAML:
//	a:
//	  b:
//	    c: 1
//
//	# Given path:
//	a.b
//
//	# Original result:
//	c: 1 node
//
//	# New result:
//	b node
//
// Features which are not used here were removed.
//
// The second major difference is that it attempts to find the closest [ast.Node] parent
// if it can't find a direct match.
// For example:
//
//	# Given YAML:
//	a:
//	  b:
//	    c: 1
//
//	# Given path:
//	a.b.d
//
//	# Original result:
//	error
//
//	# New result:
//	b node
package yamlpath
