package nobl9repo

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	v1objects "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
)

func NewRepo(client *sdk.Client) *Repo {
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

type usersResponse struct {
	Users []User `json:"users"`
}

type User struct {
	UserID string `json:"userId"`
}

func (r *Repo) GetUser(ctx context.Context, id string) (*User, error) {
	q := url.Values{"phrase": []string{id}}
	req, err := r.client.CreateRequest(ctx, http.MethodGet, "/usrmgmt/v2/users", nil, q, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var users usersResponse
	if err = json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}
	if len(users.Users) == 1 {
		return &users.Users[0], nil
	}
	return nil, nil
}

type Roles struct {
	OrganizationRoles []Role `json:"organizationRoles"`
	ProjectRoles      []Role `json:"projectRoles"`
}

type Role struct {
	Name string `json:"name"`
}

func (r *Repo) GetRoles(ctx context.Context) (*Roles, error) {
	req, err := r.client.CreateRequest(ctx, http.MethodGet, "/usrmgmt/v2/users/search-filters", nil, nil, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var roles Roles
	if err = json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, err
	}
	return &roles, nil
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
