//+build dev

package stripe

import "net/http"

// FS exports the filesystem
var CertsFS http.FileSystem = http.Dir("../../data/certs")
