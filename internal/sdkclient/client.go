package sdkclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	v1objects "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
)

type Client struct {
	client *sdk.Client
}

func New(client *sdk.Client) *Client {
	return &Client{client: client}
}

func (c *Client) GetObject(ctx context.Context, kind manifest.Kind, name, project string) (manifest.Object, error) {
	header := http.Header{}
	if project != "" {
		header.Set(sdk.HeaderProject, project)
	} else {
		header.Set(sdk.HeaderProject, sdk.ProjectsWildcard)
	}
	objects, err := c.client.Objects().V1().Get(
		ctx,
		kind,
		header,
		url.Values{
			v1objects.QueryKeyName: []string{name},
		},
	)
	if err != nil {
		return nil, err
	}
	if len(objects) == 0 {
		return nil, nil
	}
	return objects[0], nil
}
