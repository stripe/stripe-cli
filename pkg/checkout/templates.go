//go:generate go run -tags=dev gen_static.go

package checkout

import (
	"html/template"
	"io/ioutil"
)

// RedirectTemplate returns the template for the initial checkout redirect
func RedirectTemplate() (*template.Template, error) {
	return parseTemplate("html/redirect.html")
	//tmpl := template.Must(template.Parse(filedata))
}

// SuccessTemplate returns the template for the initial checkout redirect
func SuccessTemplate() (*template.Template, error) {
	return parseTemplate("html/success.html")
}

// CancelTemplate returns the template for the initial checkout redirect
func CancelTemplate() (*template.Template, error) {
	return parseTemplate("html/cancel.html")
}

func parseTemplate(fileName string) (*template.Template, error) {
	file, err := FS.Open(fileName)
	if err != nil {
		return nil, err
	}

	filedata, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	tmpl := template.New(fileName)
	tmpl.Parse(string(filedata))
	return tmpl, nil
}
