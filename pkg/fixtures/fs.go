//+build dev

package fixtures

import "net/http"

var FS http.FileSystem = http.Dir("../../triggers")
