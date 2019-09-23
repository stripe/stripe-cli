// This file is generated; DO NOT EDIT.

package cmd


func addEventsToListenCmd(cmd *listenCmd) {
	{{ range $_, $nsName := .Events }}
	cmd.validEvents["{{ $nsName }}"] = true {{end}}
}



