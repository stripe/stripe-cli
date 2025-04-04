// This file is generated; DO NOT EDIT.

package proxy

var validEvents = map[string]bool{ {{ range $_, $nsName := .Events }}
"{{ $nsName }}": true, {{end}}
}

var validThinEvents = map[string]bool{ {{ range $_, $nsName := .ThinEvents }}
"{{ $nsName }}": true, {{end}}
}

var validPreviewEvents = map[string]bool{ {{ range $_, $nsName := .PreviewEvents }}
"{{ $nsName }}": true, {{end}}
}
