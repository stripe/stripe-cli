package terminal

// ReaderData contains information about a specific Stripe compatible reader. An example is the Verifone P400.
type ReaderData struct {
	Name        string
	URL         string
	Description string
}

var readerVerifoneP400 = &ReaderData{
	Name:        "Verifone P400",
	Description: "The Verifone P400 is a countertop reader for web-based Stripe Terminal apps. It connects to the Stripe Terminal SDK over the internet. This reader is compatible with JavaScript SDK.",
	URL:         "https://www.verifone.com/sites/default/files/2018-01/p400_datasheet_ltr_013018.pdf",
}

// ReaderList is a map containing all of the Stripe compatible reader types that we support in the CLI.
var ReaderList = map[string]*ReaderData{
	"verifone-p400": readerVerifoneP400,
}

// ReaderNames is a function that uses ReaderList to extract the human friendly names of the CLI supported readers.
// it returns a map of the human friendly reader names as strings
func ReaderNames() []string {
	names := make([]string, 0, len(ReaderList))
	for index := range ReaderList {
		names = append(names, ReaderList[index].Name)
	}

	return names
}
