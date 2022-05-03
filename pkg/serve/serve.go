package serve

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"
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
type SubFileSystem struct {
	FileSystem http.Dir
	Dir        string
	// dir http.Dir
}

// Open is a wrapper around the Open method of the Dir FS
func (fsys SubFileSystem) Open(name string) (http.File, error) {
	// Here we want to handle the additional case for windows where providing the drive with the absolute path allows
	// the visitor to escape the intended directory in the File Server
	baseDir := fsys.Dir
	if baseDir == "" {
		baseDir = "."
	}
	fullName := filepath.Join(baseDir, filepath.FromSlash(path.Clean("/"+name)))

	relative, err := filepath.Rel(baseDir, fullName)
	fmt.Printf("Base Path: %s\n", baseDir)
	fmt.Printf("Target Path: %s\n", fullName)
	fmt.Printf("Relative Path: %s\n", relative)
	if err != nil {
		fmt.Printf("Relative Error : %s\n", err.Error())
	}

	file, err := fsys.FileSystem.Open(name)
	if err != nil {
		fmt.Printf("[Open] You do not have permissions to open %s. %v\n", name, err.Error())
		return nil, err
	}

	fmt.Printf("[Open] Opening file %s\n", name)
	return file, err
}
