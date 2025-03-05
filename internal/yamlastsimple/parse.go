package yamlastsimple

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type File struct {
	Docs []*Document
}

type Document struct {
	Offset int
	Lines  []*Line
}

func (d *Document) FindLine(lineNum int) *Line {
	if lineNum < 0 || lineNum >= len(d.Lines) {
		return nil
	}
	return d.Lines[lineNum]
}

func (d *Document) FindLineByPath(path string) *Line {
	for _, line := range d.Lines {
		if line.Path == path {
			return line
		}
	}
	return nil
}

type LineType int

const (
	LineTypeUndefined LineType = 1 << iota
	LineTypeEmpty
	LineTypeComment
	LineTypeMapping
	LineTypeList
	LineTypeBlockScalar
	LineTypeDocSeparator
)

type Line struct {
	// Path is the actual path of this line within a [Document].
	Path string
	// GeneralizedPath is a path that can be used to match the line with a given general path.
	// Example:
	//   $[0].metadata.labels -> $.metadata.labels
	//   $.metadata.labels[0].name -> $.metadata.labels[*].name
	GeneralizedPath string
	Type            LineType

	value         string
	indent        int
	valueColonIdx int
	listIndex     int
}

func (l *Line) GetIndent() int {
	return l.indent
}

// IsType returns true if the line is of the specified [LineType].
func (l *Line) IsType(typ LineType) bool {
	return l.Type&typ != 0
}

// HasMapValue returns true if the line is a mapping and has a value.
func (l *Line) HasMapValue() bool {
	return l.IsType(LineTypeMapping) && l.valueColonIdx != -1 && l.valueColonIdx+2 < len(l.value)
}

// GetKeyPos returns the start and end position of the key on the given line.
// End position is exclusive, while start is inclusive.
func (l *Line) GetKeyPos() (start, end int) {
	if !l.IsType(LineTypeMapping) && !l.IsType(LineTypeUndefined) {
		return 0, 0
	}
	if l.valueColonIdx != -1 {
		return l.indent, l.indent + l.valueColonIdx
	}
	return l.indent, l.indent + len(l.value)
}

// GetValuePos returns the start and end position of the value on the given line.
// End position is exclusive, while start is inclusive.
func (l *Line) GetValuePos() (start, end int) {
	if !l.HasMapValue() {
		return 0, 0
	}
	return l.indent + l.valueColonIdx + 2, l.indent + len(l.value)
}

// GetMapKey returns the key of the mapping line.
func (l *Line) GetMapKey() string {
	start, end := l.GetKeyPos()
	start -= l.indent
	end -= l.indent
	if start < 0 || end < 0 {
		return ""
	}
	return l.value[start:end]
}

// GetMapValue returns the value of the mapping line.
func (l *Line) GetMapValue() string {
	start, end := l.GetValuePos()
	start -= l.indent
	end -= l.indent
	if start < 0 || end < 0 {
		return ""
	}
	return l.value[start:end]
}

func (l *Line) addType(typ LineType) {
	l.Type |= typ
}

const yamlDocSeparator = "---"

// TODO: Maybe we don't need it? Maybe ParseDocument is enough.
func ParseFile(content string) *File {
	docs := make([]*Document, 0)
	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); {
		doc := parseDocument(lines[i:])
		doc.Offset = i
		docs = append(docs, doc)
		i += len(doc.Lines)
	}
	file := &File{Docs: docs}
	return file
}

func parseDocument(lines []string) *Document {
	parsedLines := parseDocumentLines(lines)
	parsedLines = postProcessPaths(parsedLines)
	return &Document{Lines: parsedLines}
}

func parseDocumentLines(lines []string) []*Line {
	parsedLines := make([]*Line, 0)

	blockScalar := false
	blockScalarIndent := 0

	for _, line := range lines {
		parsedLine := &Line{
			Type:          LineTypeUndefined,
			value:         line,
			valueColonIdx: -1,
		}
		if line == yamlDocSeparator {
			parsedLine.Type = LineTypeDocSeparator
			parsedLines = append(parsedLines, parsedLine)
			break
		}

		if blockScalar {
			if getIndentLevel(line) >= blockScalarIndent {
				parsedLine.Type = LineTypeBlockScalar
				parsedLine.indent = getIndentLevel(line)
				parsedLines = append(parsedLines, parsedLine)
				continue
			}
			blockScalar = false
		}

		indent := getIndentLevel(line)
		line = line[indent:]
		parsedLine.value = line
		parsedLine.indent = indent

		switch {
		case line == "":
			parsedLine.Type = LineTypeEmpty
		case line[0] == '#':
			parsedLine.Type = LineTypeComment
		case line[0] == '-':
			parsedLine.indent += 2
			parsedLine.Type = LineTypeList
			if parsedLine.valueColonIdx = strings.Index(line, ":"); parsedLine.valueColonIdx != -1 {
				parsedLine.Path = strings.TrimSpace(line[1:parsedLine.valueColonIdx])
				parsedLine.addType(LineTypeMapping)
			}
		default:
			if !isMappingNode(line) {
				parsedLine.Type = LineTypeUndefined
				break
			}
			if parsedLine.valueColonIdx = strings.Index(line, ":"); parsedLine.valueColonIdx != -1 {
				parsedLine.Path = strings.TrimSpace(line[:parsedLine.valueColonIdx])
				parsedLine.Type = LineTypeMapping
			}
			if isBlockScalar(line) {
				blockScalar = true
				blockScalarIndent = indent + 2
			}
		}
		parsedLines = append(parsedLines, parsedLine)
	}
	return parsedLines
}

func postProcessPaths(parsedLines []*Line) []*Line {
	for i, line := range parsedLines {
		switch {
		case line.IsType(LineTypeDocSeparator):
			continue
		case line.IsType(LineTypeComment):
			continue
		case line.IsType(LineTypeList):
			prevLine := findPreviousLine(parsedLines, i)
			if prevLine == nil {
				line.Path = "$[0]." + line.Path
				break
			}
			// Parent.
			if prevLine.indent < line.indent {
				line.Path = prevLine.Path + "[0]." + line.Path
				//prevLine.Path += "[*]"
				break
			}
			// Sibling.
			if bracketIdx := strings.LastIndex(prevLine.Path, "["); bracketIdx != -1 {
				line.listIndex = prevLine.listIndex + 1
				line.Path = prevLine.Path[:bracketIdx] + "[" + strconv.Itoa(line.listIndex) + "]." + line.Path
			}
		default:
			prevLine := findPreviousLine(parsedLines, i)
			if prevLine == nil {
				line.Path = "$." + line.Path
				break
			}
			// Parent.
			if prevLine.indent < line.indent {
				line.Path = prevLine.Path + "." + line.Path
				break
			}
			// Sibling.
			if dotIndex := strings.LastIndex(prevLine.Path, "."); dotIndex != -1 {
				line.listIndex = prevLine.listIndex
				line.Path = prevLine.Path[:dotIndex] + "." + line.Path
			}
		}
		// Dirty hack to remove trailing dots, saves up a lot of if's though...
		if len(line.Path) > 0 && line.Path[len(line.Path)-1] == '.' {
			line.Path = line.Path[:len(line.Path)-1]
		}
	}
	for _, line := range parsedLines {
		line.GeneralizedPath = generalizePath(line.Path)
	}
	return parsedLines
}

func findPreviousLine(lines []*Line, offset int) *Line {
	for i := offset - 1; i >= 0; i-- {
		if !lines[i].IsType(LineTypeMapping) && !lines[i].IsType(LineTypeList) {
			continue
		}
		// Parent.
		if lines[i].indent < lines[offset].indent {
			return lines[i]
		}
		// Sibling.
		if lines[i].indent == lines[offset].indent {
			return lines[i]
		}
	}
	return nil
}

func isBlockScalar(s string) bool {
	for i := len(s) - 1; i >= 0; i-- {
		r := rune(s[i])
		if r >= utf8.RuneSelf && !unicode.IsSpace(r) || asciiSpace[r] == 0 {
			if r == '|' || r == '>' {
				return true
			}
			return false
		}
	}
	return false
}

func isMappingNode(s string) bool {
	for _, r := range s {
		// Ignore comments.
		if r == '#' {
			return false
		}
		if r == ':' {
			return true
		}
	}
	return false
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

func getIndentLevel(s string) int {
	ctr := 0
	for _, r := range s {
		if r >= utf8.RuneSelf && !unicode.IsSpace(r) || asciiSpace[r] == 0 {
			return ctr
		}
		ctr++
	}
	return ctr
}

// generalizePath generalizes the root path of the YAML document by:
//   - removing the first array marker (if it exists)
//   - replacing list indices with wildcards '*'
func generalizePath(path string) string {
	if len(path) < 2 {
		return path
	}
	if path[0] == '$' && path[1] == '[' {
		closingBracketIndex := strings.IndexRune(path, ']')
		if closingBracketIndex != -1 {
			path = "$" + path[closingBracketIndex+1:]
		}
	}
	return replaceListIndicesWithWildcards(path)
}

func replaceListIndicesWithWildcards(path string) string {
	if len(path) == 0 {
		return path
	}
	var arrStart bool
	replaced := make([]byte, 0, len(path))
	for i := range path {
		switch {
		case path[i] == '[':
			arrStart = true
		case path[i] == ']':
			replaced = append(replaced, '*')
			arrStart = false
		case arrStart:
			continue // skip array index
		}
		replaced = append(replaced, path[i])
	}
	return string(replaced)
}
