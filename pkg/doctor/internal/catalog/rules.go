package catalog

// Topics lists every migration topic doctor knows about, keyed by the
// name used on the command line (`stripe doctor <topic> <dir>`). The
// rule itself is fetched live from the doctor_rules API — see
// judge.FetchRule — rather than kept here.
var Topics = map[string]bool{
	"dpm": true,
}
