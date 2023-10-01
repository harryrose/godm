package writer

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
)

const (
	FileType = "file"
)

func init() {
	var err error
	downloadRoot, err = os.Getwd()
	if err != nil {
		log.Fatalf("error getting current working directory: %v", err)
	}
}

var downloadRoot = ""

func ForceDownloadRoot(str string) {
	downloadRoot = str
}

func ParseFileItemFromURL(str string) (OpenWriterCloser, error) {
	u, err := url.Parse(str)
	if err != nil {
		return nil, fmt.Errorf("file url is not well formed: %w", err)
	}
	if u.Scheme != "file" {
		return nil, fmt.Errorf("expected a file:// url to be passed. got %v", str)
	}
	// by making the path absolute, when we clean the path, go will ensure the
	// cleaned path can't go higher than root.  Then, when we join this path to
	// the download root, we can guarantee that there's no break-out potential.
	//
	// For paths that are already absolute, the full path will be created *inside*
	// the download root.
	p := path.Join(u.Host, u.Path)
	if !path.IsAbs(p) {
		p = path.Join("/", p)
	}
	p = path.Clean(p)
	p = path.Join(downloadRoot, p)
	return &FileSourceConfiguration{Path: p}, nil
}

type FileSourceConfiguration struct {
	Path string
}

func (f *FileSourceConfiguration) Type() string {
	return FileType
}

func (f *FileSourceConfiguration) String() string {
	return f.Path
}

func (f *FileSourceConfiguration) OpenWriteCloser() (io.WriteCloser, error) {
	return os.Create(f.Path)
}
