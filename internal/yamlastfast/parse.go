package yamlastfast

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type File struct {
	Docs []*Document
}

func (f File) FindLine(lineNum int) *Line {
	i := 0
	for _, doc := range f.Docs {
		if line := doc.FindLine(lineNum - i); line != nil {
			return line
		}
		i += len(doc.Lines)
	}
	return nil
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
	Path   string
	Indent int
	Type   LineType
	// TODO: Instead we could keep start and end positions.
	Value string

	listIndex int
}

func (l *Line) IsType(typ LineType) bool {
	return l.Type&typ != 0
}

func (l *Line) GetMapValue() string {
	if !l.IsType(LineTypeMapping) {
		return ""
	}
	colonIdx := strings.Index(l.Value, ":")
	if colonIdx == -1 || colonIdx+1 >= len(l.Value) {
		return ""
	}
	return strings.TrimSpace(l.Value[colonIdx+1:])
}

func (l *Line) GetMapKey() string {
	if !l.IsType(LineTypeMapping) {
		return ""
	}
	colonIdx := strings.Index(l.Value, ":")
	if colonIdx == -1 {
		return ""
	}
	return strings.TrimSpace(l.Value[:colonIdx])
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
			Type:  LineTypeUndefined,
			Value: line,
		}
		if line == yamlDocSeparator {
			parsedLine.Type = LineTypeDocSeparator
			parsedLines = append(parsedLines, parsedLine)
			break
		}

		if blockScalar {
			if getIndentLevel(line) >= blockScalarIndent {
				parsedLine.Type = LineTypeBlockScalar
				parsedLine.Indent = getIndentLevel(line)
				parsedLines = append(parsedLines, parsedLine)
				continue
			}
			blockScalar = false
		}

		indent := getIndentLevel(line)
		line = line[indent:]
		parsedLine.Indent = indent

		switch {
		case line == "":
			parsedLine.Type = LineTypeEmpty
		case line[0] == '#':
			parsedLine.Type = LineTypeComment
		case line[0] == '-':
			parsedLine.Indent += 2
			parsedLine.Type = LineTypeList
			if colonIdx := strings.Index(line, ":"); colonIdx != -1 {
				parsedLine.Path = strings.TrimSpace(line[1:colonIdx])
				parsedLine.addType(LineTypeMapping)
			}
		default:
			if !isMappingNode(line) {
				parsedLine.Type = LineTypeUndefined
				break
			}
			if colonIdx := strings.Index(line, ":"); colonIdx != -1 {
				parsedLine.Path = strings.TrimSpace(line[:colonIdx])
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
			if prevLine.Indent < line.Indent {
				line.Path = prevLine.Path + "[0]." + line.Path
				prevLine.Path += "[*]"
				break
			}
			// Sibling.
			if bracketIdx := strings.LastIndex(prevLine.Path, "["); bracketIdx != -1 {
				line.Path = prevLine.Path[:bracketIdx] + "[" + strconv.Itoa(prevLine.listIndex+1) + "]." + line.Path
			}
		default:
			prevLine := findPreviousLine(parsedLines, i)
			if prevLine == nil {
				line.Path = "$." + line.Path
				break
			}
			// Parent.
			if prevLine.Indent < line.Indent {
				line.Path = prevLine.Path + "." + line.Path
				break
			}
			// Sibling.
			if dotIndex := strings.LastIndex(prevLine.Path, "."); dotIndex != -1 {
				line.Path = prevLine.Path[:dotIndex] + "." + line.Path
			}
		}
		// Dirty hack to remove trailing dots, saves up a lot of if's though...
		if len(line.Path) > 0 && line.Path[len(line.Path)-1] == '.' {
			line.Path = line.Path[:len(line.Path)-1]
		}
	}
	return parsedLines
}

func findPreviousLine(lines []*Line, offset int) *Line {
	for i := offset - 1; i >= 0; i-- {
		if !lines[i].IsType(LineTypeMapping) && !lines[i].IsType(LineTypeList) {
			continue
		}
		// Parent.
		if lines[i].Indent < lines[offset].Indent {
			return lines[i]
		}
		// Sibling.
		if lines[i].Indent == lines[offset].Indent {
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
