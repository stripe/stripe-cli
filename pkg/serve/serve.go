package serve

import (
	"fmt"
	"io/fs"
	"net/http"
)

// TestFile ccc
type TestFile struct {
	http.File
}

// Readdir ccc
func (f TestFile) Readdir(n int) (fis []fs.FileInfo, err error) {
	files, err := f.File.Readdir(n)
	for _, file := range files {
		fmt.Printf("Servable File: %v\n", file.Name())
		fis = append(fis, file)
	}
	return
}

// TestFileSystem ccc
type TestFileSystem struct {
	http.FileSystem
}

// Open ccc
func (fsys TestFileSystem) Open(name string) (http.File, error) {
	file, err := fsys.FileSystem.Open(name)
	if err != nil {
		fmt.Printf("[Open] You do not have permissions to open %s. %v\n", name, err.Error())
		return nil, err
	}

	fmt.Printf("[Open] Opening file %s\n", name)
	return TestFile{file}, err
}
