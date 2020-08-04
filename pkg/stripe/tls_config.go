//go:generate go run -tags=dev vfsgen.go

package stripe

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
)

// GetTLSConfig returns a tls.Config object that uses the bundled DigiCert CA
// certificates rather than the system's cert store.
func GetTLSConfig() (*tls.Config, error) {
	caCertPool := x509.NewCertPool()

	f, err := CertsFS.Open("/DigiCertGlobalRootCA.crt.pem")
	if err != nil {
		return nil, err
	}
	filedata, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	caCertPool.AppendCertsFromPEM(filedata)

	f, err = CertsFS.Open("/DigiCertHighAssuranceEVRootCA.crt.pem")
	if err != nil {
		return nil, err
	}
	filedata, err = ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	caCertPool.AppendCertsFromPEM(filedata)

	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}
