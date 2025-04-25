package files

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/nobl9/nobl9-go/manifest"
	v1alphaAgent "github.com/nobl9/nobl9-go/manifest/v1alpha/agent"
	v1alphaAlert "github.com/nobl9/nobl9-go/manifest/v1alpha/alert"
	v1alphaAlertMethod "github.com/nobl9/nobl9-go/manifest/v1alpha/alertmethod"
	v1alphaAlertPolicy "github.com/nobl9/nobl9-go/manifest/v1alpha/alertpolicy"
	v1alphaAlertSilence "github.com/nobl9/nobl9-go/manifest/v1alpha/alertsilence"
	v1alphaAnnotation "github.com/nobl9/nobl9-go/manifest/v1alpha/annotation"
	v1alphaBudgetAdjustment "github.com/nobl9/nobl9-go/manifest/v1alpha/budgetadjustment"
	v1alphaDataExport "github.com/nobl9/nobl9-go/manifest/v1alpha/dataexport"
	v1alphaDirect "github.com/nobl9/nobl9-go/manifest/v1alpha/direct"
	v1alphaProject "github.com/nobl9/nobl9-go/manifest/v1alpha/project"
	v1alphaReport "github.com/nobl9/nobl9-go/manifest/v1alpha/report"
	v1alphaRoleBinding "github.com/nobl9/nobl9-go/manifest/v1alpha/rolebinding"
	v1alphaService "github.com/nobl9/nobl9-go/manifest/v1alpha/service"
	v1alphaSLO "github.com/nobl9/nobl9-go/manifest/v1alpha/slo"
	v1alphaUserGroup "github.com/nobl9/nobl9-go/manifest/v1alpha/usergroup"
	"github.com/pkg/errors"

	"github.com/nobl9/nobl9-language-server/internal/yamlast"
)

// ObjectNode is a wrapper over single [manifest.Object] which holds both the object and its [yamlast.Node].
type ObjectNode struct {
	Kind    manifest.Kind
	Version manifest.Version
	Object  manifest.Object
	Node    *yamlast.Node
	// Err is the error that occurred while decoding the [manifest.Object] (if any).
	Err error
}

func (o *ObjectNode) copy() *ObjectNode {
	return &ObjectNode{
		Kind:    o.Kind,
		Version: o.Version,
		Object:  o.Object,
		Node:    o.Node,
		Err:     o.Err,
	}
}

func parseObjectNode(ctx context.Context, node *yamlast.Node) *ObjectNode {
	object := &ObjectNode{Node: node}
	object.Version, object.Kind, object.Err = inferObjectVersionAndKind(node.Node)
	if object.Err == nil {
		object.Object, object.Err = parseObject(ctx, object)
	}
	return object
}

func parseObject(ctx context.Context, object *ObjectNode) (manifest.Object, error) {
	var buf bytes.Buffer
	dec := yaml.NewDecoder(&buf, yaml.Strict())
	manifestObject, err := decodeObject(object.Version, object.Kind, func(v any) error {
		return dec.DecodeFromNodeContext(ctx, object.Node.Node, v)
	})
	if err != nil {
		return nil, err
	}
	return manifestObject, nil
}

type decodeFunc func(v any) error

func decodeObject(version manifest.Version, kind manifest.Kind, decode decodeFunc) (manifest.Object, error) {
	switch version {
	case manifest.VersionV1alpha:
		return decodeV1alphaObject(kind, decode)
	default:
		return nil, fmt.Errorf("%s is %w", version, manifest.ErrInvalidVersion)
	}
}

func decodeV1alphaObject(kind manifest.Kind, decode decodeFunc) (manifest.Object, error) {
	//exhaustive:enforce
	switch kind {
	case manifest.KindService:
		return genericParseObject[v1alphaService.Service](decode)
	case manifest.KindSLO:
		return genericParseObject[v1alphaSLO.SLO](decode)
	case manifest.KindProject:
		return genericParseObject[v1alphaProject.Project](decode)
	case manifest.KindAgent:
		return genericParseObject[v1alphaAgent.Agent](decode)
	case manifest.KindDirect:
		return genericParseObject[v1alphaDirect.Direct](decode)
	case manifest.KindAlert:
		return genericParseObject[v1alphaAlert.Alert](decode)
	case manifest.KindAlertMethod:
		return genericParseObject[v1alphaAlertMethod.AlertMethod](decode)
	case manifest.KindAlertPolicy:
		return genericParseObject[v1alphaAlertPolicy.AlertPolicy](decode)
	case manifest.KindAlertSilence:
		return genericParseObject[v1alphaAlertSilence.AlertSilence](decode)
	case manifest.KindRoleBinding:
		return genericParseObject[v1alphaRoleBinding.RoleBinding](decode)
	case manifest.KindDataExport:
		return genericParseObject[v1alphaDataExport.DataExport](decode)
	case manifest.KindAnnotation:
		return genericParseObject[v1alphaAnnotation.Annotation](decode)
	case manifest.KindUserGroup:
		return genericParseObject[v1alphaUserGroup.UserGroup](decode)
	case manifest.KindBudgetAdjustment:
		return genericParseObject[v1alphaBudgetAdjustment.BudgetAdjustment](decode)
	case manifest.KindReport:
		return genericParseObject[v1alphaReport.Report](decode)
	default:
		return nil, fmt.Errorf("%s is %w", kind, manifest.ErrInvalidKind)
	}
}

func genericParseObject[T manifest.Object](decode decodeFunc) (T, error) {
	var object T
	if err := decode(&object); err != nil {
		return object, err
	}
	return object, nil
}

func inferObjectVersionAndKind(node ast.Node) (version manifest.Version, kind manifest.Kind, err error) {
	mn, ok := node.(*ast.MappingNode)
	if !ok {
		return version, kind, errors.New("object cannot be parsed")
	}
	for _, v := range mn.Values {
		key, ok := v.Key.(*ast.StringNode)
		if !ok {
			continue
		}
		if key.Value != "apiVersion" && key.Value != "kind" {
			continue
		}
		value, ok := v.Value.(*ast.StringNode)
		if !ok {
			continue
		}
		var err error
		switch key.Value {
		case "apiVersion":
			version, err = manifest.ParseVersion(value.Value)
		case "kind":
			kind, err = manifest.ParseKind(value.Value)
		}
		if err != nil {
			return version, kind, err
		}
	}
	return version, kind, nil
}
