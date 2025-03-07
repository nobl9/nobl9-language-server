package nobl9repo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	v1objects "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"

	"github.com/nobl9/nobl9-language-server/internal/version"
)

const envPrefix = "NOBL9_LANGUAGE_SERVER_"

func NewRepo() (*Repo, error) {
	options := []sdk.ConfigOption{
		sdk.ConfigOptionEnvPrefix(envPrefix),
	}
	conf, err := sdk.ReadConfig(options...)
	if err != nil {
		return nil, err
	}
	client, err := sdk.NewClient(conf)
	if err != nil {
		return nil, err
	}
	client.SetUserAgent(version.GetUserAgent())
	return &Repo{
		client: client,
		cache:  newDataCache(),
	}, nil
}

type Repo struct {
	cache  *dataCache
	client *sdk.Client
}

func (r *Repo) GetDefaultProject() string {
	return r.client.Config.Project
}

func (r *Repo) Apply(ctx context.Context, objects []manifest.Object) error {
	return r.client.Objects().V1().Apply(ctx, objects)
}

func (r *Repo) Delete(ctx context.Context, objects []manifest.Object) error {
	return r.client.Objects().V1().Delete(ctx, objects)
}

func (r *Repo) GetAllNames(ctx context.Context, kind manifest.Kind, project string) ([]string, error) {
	cacheKey := fmt.Sprintf("GetAllNames:%s:%s", kind, project)
	if data, ok := r.cache.Get(ctx, cacheKey); ok {
		names, _ := data.([]string)
		return names, nil
	}

	header := http.Header{}
	if project != "" {
		header.Set(sdk.HeaderProject, project)
	}
	objects, err := r.client.Objects().V1().Get(
		ctx,
		kind,
		header,
		url.Values{},
	)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(objects))
	for _, obj := range objects {
		names = append(names, obj.GetName())
	}
	r.cache.Put(cacheKey, names)
	return names, nil
}

func (r *Repo) GetObject(ctx context.Context, kind manifest.Kind, name, project string) (manifest.Object, error) {
	cacheKey := fmt.Sprintf("GetObject:%s:%s:%s", kind, name, project)
	if data, ok := r.cache.Get(ctx, cacheKey); ok {
		object, _ := data.(manifest.Object)
		return object, nil
	}

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
		r.cache.Put(cacheKey, nil)
		return nil, nil
	}
	r.cache.Put(cacheKey, objects[0])
	return objects[0], nil
}

type usersResponse struct {
	Users []*User `json:"users"`
}

type User struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

func (r *Repo) GetUser(ctx context.Context, id string) (*User, error) {
	cacheKey := fmt.Sprintf("GetUser:%s", id)
	if data, ok := r.cache.Get(ctx, cacheKey); ok {
		user, _ := data.(*User)
		return user, nil
	}

	users, err := r.GetUsers(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(users) == 1 {
		r.cache.Put(cacheKey, users[0])
		return users[0], nil
	}
	r.cache.Put(cacheKey, nil)
	return nil, nil
}

func (r *Repo) GetUsers(ctx context.Context, phrase string) ([]*User, error) {
	q := url.Values{"phrase": []string{phrase}}
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
	return users.Users, nil
}

type Roles struct {
	OrganizationRoles []Role `json:"organizationRoles"`
	ProjectRoles      []Role `json:"projectRoles"`
}

type Role struct {
	Name string `json:"name"`
}

func (r *Repo) GetRoles(ctx context.Context) (*Roles, error) {
	cacheKey := "GetRoles"
	if data, ok := r.cache.Get(ctx, cacheKey); ok {
		roles, _ := data.(*Roles)
		return roles, nil
	}

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
	r.cache.Put(cacheKey, &roles)
	return &roles, nil
}
