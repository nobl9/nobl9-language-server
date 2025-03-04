package yamlpath

import (
	"strings"
)

// Match returns true if the yaml path representation matches concrete path.
// The path can contain special identifiers, like wildcards.
func Match(yamlPath, concrete string) bool {
	if yamlPath == concrete {
		return true
	}
	ys := splitPath(yamlPath)
	cs := splitPath(concrete)

	if len(ys) != len(cs) {
		return false
	}
	for i := range ys {
		if ys[i] == cs[i] || ys[i] == "*" || ys[i] == "~" {
			continue
		}
		if ys[i] == "[*]" && isSubscript(cs[i]) {
			continue
		}
		return false
	}
	return true
}

func isSubscript(s string) bool {
	return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
}

// splitPath splits the path into parts based on path separators '.' and '[]'.
func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	n := strings.Count(path, ".") + strings.Count(path, "[") + 1
	if n > len(path)+1 {
		n = len(path) + 1
	}
	a := make([]string, n)
	n--
	i := 0
	for i < n {
		var m int
		for m = range path {
			if path[m] == '.' || path[m] == '[' {
				break
			}
		}
		if m < 0 {
			break
		}
		a[i] = path[:m]
		if path[m-1] == ']' {
			a[i] = "[" + a[i]
		}
		path = path[m+1:]
		i++
	}
	a[i] = path
	if path[len(path)-1] == ']' {
		a[i] = "[" + a[i]
	}
	return a[:i+1]
}
