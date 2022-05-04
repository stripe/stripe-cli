package serve

import (
	"io/fs"
	"net/http"
	"strings"
)

// DirWrapper wrapps the http.Dir implementation of the http.FileSystem interface so we can add custom logic
type DirWrapper struct {
	http.Dir
}

// Open wraps the http.Dir implementation and adds an additional layer of error classification
func (fsys DirWrapper) Open(name string) (http.File, error) {
	file, err := fsys.Dir.Open(name)

	if (err != nil) && strings.Contains(err.Error(), "filename, directory name, or volume label syntax is incorrect") {
		return file, fs.ErrInvalid
	}

	return file, err
}
