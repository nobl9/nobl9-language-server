package yamlpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_normalizeRootArrayPath(t *testing.T) {
	tests := []struct {
		Input    string
		Expected string
	}{
		{Input: "$", Expected: "$"},
		{Input: "$[1]", Expected: "$"},
		{Input: "$[10]", Expected: "$"},
		{Input: "$.[3]", Expected: "$.[3]"},
		{Input: "$[5].A.B", Expected: "$.A.B"},
	}
	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			assert.Equal(t, test.Expected, NormalizeRootPath(test.Input))
		})
	}
}
