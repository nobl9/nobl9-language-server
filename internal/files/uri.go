package files

import (
	"fmt"
	"net/url"
	"strings"
)

type URI = string

func filePathFromURI(uri URI) (string, error) {
	parsedURL, err := url.ParseRequestURI(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI %v: %w", uri, err)
	}
	if parsedURL.Scheme != "file" {
		return "", fmt.Errorf("only file URIs are supported, got %v", parsedURL.Scheme)
	}
	return strings.TrimPrefix(uri, "file://"), nil
}
