//+build dev

package spec

import "net/http"

var FS http.FileSystem = http.Dir("../../api/openapi-spec")
