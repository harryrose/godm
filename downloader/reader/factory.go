package reader

import (
	"fmt"
	"io"
	"strings"
)

type OpenReadCloser interface {
	OpenReadCloser() (io.ReadCloser, int64, error)
}

func BuildFromURL(uri string) (OpenReadCloser, error) {
	colon := strings.Index(uri, ":")
	if colon < 0 {
		return nil, fmt.Errorf("url is not well formed. scheme is missing")
	}
	scheme := strings.ToLower(uri[:colon])

	cons := factory[scheme]
	if cons == nil {
		return nil, fmt.Errorf("there is no handler for type %v", scheme)
	}
	return cons(uri)
}

var factory = map[string]func(uri string) (OpenReadCloser, error){
	"http":  ParseHTTPItemFromURL,
	"https": ParseHTTPItemFromURL,
}
