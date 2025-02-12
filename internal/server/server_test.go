package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"

	v1alphaParser "github.com/nobl9/nobl9-go/manifest/v1alpha/parser"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/service"
)

func Test_Handlers_Diagnostics(t *testing.T) {
	v1alphaParser.UseStrictDecodingMode = true
	text := `
apiVersion: n9/v1alpha
metadata:
  name: test
  project: default
spec:
  description: Test service`

	var svc service.Service
	err := yaml.NewDecoder(strings.NewReader(text),
		yaml.DisallowDuplicateKey(),
		yaml.DisallowUnknownField(),
	).Decode(&svc)
	if syntaxErr := yaml.AsTokenScopedError(err); syntaxErr != nil {
		fmt.Println(syntaxErr)
	}
}
