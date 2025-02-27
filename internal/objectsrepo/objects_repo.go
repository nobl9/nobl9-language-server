package objectsrepo

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	v1objects "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
)

func NewObjectsRepo(client *sdk.Client) *Repo {
	return &Repo{
		client:  client,
		objects: make(map[manifest.Kind]map[objectProject][]objectName),
	}
}

type Repo struct {
	objects map[manifest.Kind]map[objectProject][]objectName
	client  *sdk.Client
	once    sync.Once
}

type (
	objectName    = string
	objectProject = string
)

func (r *Repo) GetDefaultProject() string {
	return r.client.Config.Project
}

func (r *Repo) Apply(ctx context.Context, objects []manifest.Object) error {
	return r.client.Objects().V1().Apply(ctx, objects)
}

func (r *Repo) Delete(ctx context.Context, objects []manifest.Object) error {
	return r.client.Objects().V1().Delete(ctx, objects)
}

func (r *Repo) GetAllNames(ctx context.Context, kind manifest.Kind, project string) []string {
	r.init(ctx)
	return r.objects[kind][project]
}

func (r *Repo) GetObject(ctx context.Context, kind manifest.Kind, name, project string) (manifest.Object, error) {
	header := http.Header{}
	if project != "" {
		header.Set(sdk.HeaderProject, project)
	} else {
		header.Set(sdk.HeaderProject, sdk.ProjectsWildcard)
	}
	objects, err := r.client.Objects().V1().Get(
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

func (r *Repo) init(ctx context.Context) {
	r.once.Do(func() {
		projects, err := r.client.Objects().V1().GetV1alphaProjects(ctx, v1objects.GetProjectsRequest{})
		if err != nil {
			slog.Error("failed to fetch projects", slog.Any("error", err))
		}
		r.objects[manifest.KindProject] = map[objectProject][]objectName{
			"": make([]objectName, 0, len(projects)),
		}
		for _, project := range projects {
			r.objects[manifest.KindProject][""] = append(
				r.objects[manifest.KindProject][""],
				project.GetName(),
			)
		}
	})
}
