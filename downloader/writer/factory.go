package writer

import (
	"fmt"
	"io"
	"strings"
)

type OpenWriterCloser interface {
	OpenWriteCloser() (io.WriteCloser, error)
}

func BuildFromURL(uri string) (OpenWriterCloser, error) {
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

var factory = map[string]func(uri string) (OpenWriterCloser, error){
	"file": ParseFileItemFromURL,
}
