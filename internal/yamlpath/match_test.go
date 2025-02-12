package yamlpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		yamlPath    string
		concrete    string
		shouldMatch bool
	}{
		{
			yamlPath:    "",
			concrete:    "",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.c",
			concrete:    "a.b.c",
			shouldMatch: true,
		},
		{
			yamlPath: "a.b.c",
			concrete: "a.b.d",
		},
		{
			yamlPath:    "a.b.*",
			concrete:    "a.b.c",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.*[*]",
			concrete:    "a.b.c[1]",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.*[*][*]",
			concrete:    "a.b.c[1][10]",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.*[*][*].*",
			concrete:    "a.b.c[1][10].d",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.*[*][*].d",
			concrete:    "a.b.c[1][10].d",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.*[*][*].~",
			concrete:    "a.b.c[1][10].d",
			shouldMatch: true,
		},
	}
	for _, test := range tests {
		t.Run(test.yamlPath+"="+test.concrete, func(t *testing.T) {
			match := Match(test.yamlPath, test.concrete)
			assert.Equal(t, test.shouldMatch, match)
		})
	}
}
