package server

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	v1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
)

type objectsRepo struct {
	objects map[manifest.Kind][]objectMetadata
	client  *sdk.Client
	once    sync.Once
}

type objectMetadata struct {
	Name    string
	Project string
}

func newObjectsRepo(client *sdk.Client) *objectsRepo {
	return &objectsRepo{
		client:  client,
		objects: make(map[manifest.Kind][]objectMetadata),
	}
}

func (r *objectsRepo) GetAll(kind manifest.Kind) []objectMetadata {
	r.init()
	return r.objects[kind]
}

func (r *objectsRepo) init() {
	r.once.Do(func() {
		ctx := context.Background()
		projects, err := r.client.Objects().V1().GetV1alphaProjects(ctx, v1.GetProjectsRequest{})
		if err != nil {
			slog.Error("failed to fetch projects", slog.Any("error", err))
		}
		r.objects[manifest.KindProject] = make([]objectMetadata, 0, len(projects))
		for _, project := range projects {
			r.objects[manifest.KindProject] = append(
				r.objects[manifest.KindProject],
				objectMetadata{Name: project.GetName()})
		}
	})
}
