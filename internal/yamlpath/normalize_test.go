package yamlpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeRootPath(t *testing.T) {
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

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		Input    string
		Expected string
	}{
		{Input: "", Expected: ""},
		{Input: "$", Expected: "$"},
		{Input: "$[1]", Expected: "$[*]"},
		{Input: "$[10]", Expected: "$[*]"},
		{Input: "$.[1]", Expected: "$.[*]"},
		{Input: "$.[10]", Expected: "$.[*]"},
		{Input: "$.A[1].B[3]", Expected: "$.A[*].B[*]"},
		{Input: "$.A[10].B[300]", Expected: "$.A[*].B[*]"},
	}
	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			assert.Equal(t, test.Expected, NormalizePath(test.Input))
		})
	}
}
