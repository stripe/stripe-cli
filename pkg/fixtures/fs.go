//+build dev

package fixtures

import "net/http"

// FS exports the filesystem
var FS http.FileSystem = http.Dir("../../triggers")
