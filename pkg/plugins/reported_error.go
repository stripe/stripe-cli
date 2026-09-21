package plugins

import "errors"

// pluginReportedError marks an error that the plugin process itself produced after
// it had started, and so has already written to the terminal.
//
// Run returns errors from two distinct situations, and the caller has to tell them
// apart. A plugin that started and then failed has printed its own message, so
// printing it again would duplicate it. Everything else -- an install that failed,
// a plugin refused for being too old to read the config file, a handshake that
// never completed -- happens before the plugin is launched, so nothing has been
// printed and the caller is the only one that can say what went wrong.
type pluginReportedError struct {
	err error
}

func (e pluginReportedError) Error() string {
	return e.err.Error()
}

func (e pluginReportedError) Unwrap() error {
	return e.err
}

// PluginAlreadyReported reports whether err came from a plugin process that had
// already started, meaning the plugin has printed the message itself.
//
// A caller that exits on error must print any error for which this is false, or
// the command fails with no output at all.
func PluginAlreadyReported(err error) bool {
	var reported pluginReportedError
	return errors.As(err, &reported)
}
