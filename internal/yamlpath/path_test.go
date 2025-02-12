package yamlpath

import (
	"io"
	"testing"

	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func builder() *PathBuilder { return &PathBuilder{} }

func TestPath(t *testing.T) {
	yml := `
store:
  book:
    - author: john
      price: 10
    - author: ken
      price: 12
  bicycle:
    color: red
    price: 19.95
  bicycle*unicycle:
    price: 20.25
`
	tests := []struct {
		name       string
		path       *Path
		expected   token.Position
		firstMatch bool
	}{
		{
			name:       "$",
			path:       builder().Root().Build(),
			expected:   token.Position{Line: 2, Offset: 7},
			firstMatch: true,
		},
		{
			name:       "$.store.book",
			path:       builder().Root().Child("store").Child("book").Build(),
			expected:   token.Position{Line: 3, Offset: 15},
			firstMatch: true,
		},
		{
			name:       "$.store.book[0].author",
			path:       builder().Root().Child("store").Child("book").Index(0).Child("author").Build(),
			expected:   token.Position{Line: 4, Offset: 29},
			firstMatch: true,
		},
		{
			name:       "$.store.book[1].price",
			path:       builder().Root().Child("store").Child("book").Index(1).Child("price").Build(),
			expected:   token.Position{Line: 7, Offset: 81},
			firstMatch: true,
		},
		{
			name:       "$.store.book[0]",
			path:       builder().Root().Child("store").Child("book").Index(0).Build(),
			expected:   token.Position{Line: 4, Offset: 29},
			firstMatch: true,
		},
		{
			name:       "$.store.bicycle.price",
			path:       builder().Root().Child("store").Child("bicycle").Child("price").Build(),
			expected:   token.Position{Line: 10, Offset: 121},
			firstMatch: true,
		},
		{
			name:       `$.store.'bicycle*unicycle'.price`,
			path:       builder().Root().Child("store").Child(`bicycle*unicycle`).Child("price").Build(),
			expected:   token.Position{Line: 12, Offset: 158},
			firstMatch: true,
		},
		{
			name:     "$.store.videos",
			path:     builder().Root().Child("store").Child("videos").Build(),
			expected: token.Position{Line: 2, Offset: 7},
		},
		{
			name:     "$.shop",
			path:     builder().Root().Child("shop").Build(),
			expected: token.Position{Line: 2, Offset: 7},
		},
		{
			name:     "$.store.book[1].tag",
			path:     builder().Root().Child("store").Child("book").Index(1).Child("tag").Build(),
			expected: token.Position{Line: 6, Offset: 64},
		},
	}
	t.Run("FromString", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				path, err := FromString(test.name)
				require.NoError(t, err)
				assert.Equal(t, test.name, path.String())
			})
		}
	})
	t.Run("FilterFile", func(t *testing.T) {
		astFile, err := parser.ParseBytes([]byte(yml), 0)
		require.NoError(t, err)
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				node, firstMatch, err := test.path.FilterFile(astFile)
				require.NoError(t, err)
				pos := node.GetToken().Position
				assert.Equal(t, test.expected.Line, pos.Line)
				assert.Equal(t, test.expected.Offset, pos.Offset)
				assert.Equal(t, test.firstMatch, firstMatch)
			})
		}
	})
}

func TestPath_ReservedKeyword(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		src      string
		expected string
		failure  bool
	}{
		{
			name: "quoted path",
			path: `$.'a.b.c'.foo`,
			src: `
a.b.c:
  foo: bar
`,
			expected: "  foo: bar",
		},
		{
			name:     "contains quoted key",
			path:     `$.a'b`,
			src:      `a'b: 10`,
			expected: "a'b: 10",
		},
		{
			name:     "escaped quote",
			path:     `$.'alice\'s age'`,
			src:      `alice's age: 10`,
			expected: "alice's age: 10",
		},
		{
			name:     "unquoted white space",
			path:     `$.a  b`,
			src:      `a  b: 10`,
			expected: "a  b: 10",
		},
		{
			name:    "empty quoted key",
			path:    `$.''`,
			src:     `a: 10`,
			failure: true,
		},
		{
			name:    "unterminated quote",
			path:    `$.'foo`,
			src:     `foo: 10`,
			failure: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, err := FromString(test.path)
			if test.failure {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}
			file, err := parser.ParseBytes([]byte(test.src), 0)
			require.NoError(t, err)
			node, _, err := path.FilterFile(file)
			require.NoError(t, err)
			data, err := io.ReadAll(node)
			require.NoError(t, err)
			assert.Equal(t, test.expected, string(data))
		})
	}
}
