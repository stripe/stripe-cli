package serve

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// TestFile ccc
// type TestFile struct {
// 	http.File
// }

// // Readdir ccc
// func (f TestFile) Readdir(n int) (fis []fs.FileInfo, err error) {
// 	// This implementation of the Readdir should return what the net/http File deems "eligible"
// 	files, err := f.File.Readdir(n)
// 	for _, file := range files {
// 		fmt.Printf("Servable File: %v\n", file.Name())

// 		path.Clean("/"+dir)
// 		fis = append(fis, file)
// 	}
// 	return
// }

// SubFileSystem only allows access to the directory and its contents
type SubFileSystem string

// Open is a wrapper around the Open method of the Dir FS
func (d SubFileSystem) Open(name string) (http.File, error) {
	// Here we want to handle the additional case for windows where providing the drive with the absolute path allows
	// the visitor to escape the intended directory in the File Server
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("http: invalid character in file path")
	}
	dir := string(d)
	if dir == "" {
		dir = "."
	}
	fullName := filepath.Join(dir, filepath.FromSlash(filepath.Clean("/"+name)))
	f, err := os.Open(fullName)
	if err != nil {
		return nil, err
	}
	return f, nil
}
