//+build vfsgen

package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/shurcooL/vfsgen"

	"github.com/stripe/stripe-cli/pkg/fixtures"
)

func main() {
	// Override all file mod times to be the Unix epoch using modTimeFS.
	// We do this because vfsgen includes mod times in the output file it
	// generates, but mod times are not managed by git. As a result, they can
	// be different depending on the developer's machine. Overriding them to a
	// fixed value ensures that the generated output will only change when the
	// contents of the embedded files change.
	var fixturesInputFS http.FileSystem = modTimeFS{
		fs: fixtures.FS,
	}
	err := vfsgen.Generate(fixturesInputFS, vfsgen.Options{
		PackageName:  "fixtures",
		BuildTags:    "!dev",
		VariableName: "FS",
	})
	if err != nil {
		log.Fatalln(err)
	}
}

// modTimeFS is an http.FileSystem wrapper that modifies
// underlying fs such that all of its file mod times are set to the Unix epoch.
type modTimeFS struct {
	fs http.FileSystem
}

func (fs modTimeFS) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return modTimeFile{f}, nil
}

type modTimeFile struct {
	http.File
}

func (f modTimeFile) Stat() (os.FileInfo, error) {
	fi, err := f.File.Stat()
	if err != nil {
		return nil, err
	}
	return modTimeFileInfo{fi}, nil
}

type modTimeFileInfo struct {
	os.FileInfo
}

func (modTimeFileInfo) ModTime() time.Time {
	return time.Unix(0, 0)
}
