//+build dev

package checkout

import "net/http"

// FS exports the filesystem
var FS http.FileSystem = http.Dir("./static")
