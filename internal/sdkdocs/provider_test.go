package sdkdocs

import (
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocs_GetProperty(t *testing.T) {
	docs, err := New()
	require.NoError(t, err)

	tests := []struct {
		kind     manifest.Kind
		path     string
		expected string
	}{
		{
			kind:     manifest.KindSLO,
			path:     "$.spec.indicator.metricSource.name",
			expected: "$.spec.indicator.metricSource.name",
		},
		{
			kind:     manifest.KindService,
			path:     "$.metadata.labels",
			expected: "$.metadata.labels",
		},
		{
			kind:     manifest.KindService,
			path:     "$.metadata.labels.team",
			expected: "$.metadata.labels.~",
		},
		{
			kind:     manifest.KindService,
			path:     "$.metadata.labels.team[2]",
			expected: "$.metadata.labels.*[*]",
		},
	}

	for _, test := range tests {
		t.Run(test.kind.String()+"/"+test.path, func(t *testing.T) {
			doc := docs.GetProperty(test.kind, test.path)
			require.NotNil(t, doc)
			assert.Equal(t, test.expected, doc.Path)
		})
	}
}
