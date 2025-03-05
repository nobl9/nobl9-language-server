package nobl9repo

import "net/http"

func newResponseCache(client *http.Client) http.RoundTripper {
	return responseCache{client: client}
}

type responseCache struct {
	client *http.Client
}

func (r responseCache) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.client.Do(req)
}
