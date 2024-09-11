package proxy

// Hard-coded for now, we should autogen this when V2 event types
// are available from the Open API spec
var validThinEvents = map[string]bool{
	"*": true,
	"v1.billing.meter.error_report_triggered": true,
	"v1.billing.meter.no_meter_found":         true,
}
