// This file is generated; DO NOT EDIT.

package cmd

var validEvents = map[string]bool{ {{ range $_, $nsName := .Events }}
"{{ $nsName }}": true, {{end}}
}


