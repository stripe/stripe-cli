package proxy

// Hard-coded for now, we should autogen this when V2 event types
// are available from the Open API spec
var validV2Events = map[string]bool{
	"*": true,
	"v2.billing.meter.error_report_triggered": true,
}
