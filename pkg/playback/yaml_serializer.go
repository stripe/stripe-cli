package playback

import (
	"net/http"
	"net/url"

	"gopkg.in/yaml.v2"
)

// YAMLSerializer encodes/persists cassettes to files and decodes yaml cassette files.
type YAMLSerializer struct{}

// YAMLRequest is a playback.httpRequest interface that can be encoded to YAML
type YAMLRequest struct {
	Method  string      `yaml:"method"`
	Body    string      `yaml:"body"`
	Headers http.Header `yaml:"headers"`
	URL     url.URL     `yaml:"url"`
}

// YAMLResponse is a playback.httpRequest interface that can be encoded to YAML
type YAMLResponse struct {
	Headers    http.Header `yaml:"headers"`
	Body       string      `yaml:"body"`
	StatusCode int         `yaml:"status_code"`
}

// -------- SERIALIZATION

// serializeReq takes in an httpRequest and returns a []byte representing YAML
// the []byte can be stringified to print the actual human-readable YAML
func (s YAMLSerializer) serializeReq(input interface{}) ([]byte, error) {
	req := input.(httpRequest)
	yml := YAMLRequest{
		req.Method,
		string(req.Body),
		req.Headers,
		req.URL,
	}
	return yaml.Marshal(yml)
}

// serializeResp takes in an httpResponse and returns a []byte representing YAML
// the []byte can be stringified to print the actual human-readable YAML
func (s YAMLSerializer) serializeResp(input interface{}) ([]byte, error) {
	res := input.(httpResponse)
	yml := YAMLResponse{
		res.Headers,
		string(res.Body),
		res.StatusCode,
	}
	return yaml.Marshal(yml)
}

// -------- DESERIALIZATION
// deserializeReq takes the []byte representing YAML and returns an httpRequest
func (s YAMLSerializer) deserializeReq(data []byte) (httpRequest, error) {
	var req httpRequest

	var yamlReq YAMLRequest
	err := yaml.Unmarshal(data, &yamlReq)
	if err != nil {
		return req, err
	}

	req.Method = yamlReq.Method
	req.Body = []byte(yamlReq.Body)
	req.Headers = yamlReq.Headers
	req.URL = yamlReq.URL
	return req, nil
}

// deserializeReq takes the []byte representing YAML and returns an httpResponse
func (s YAMLSerializer) deserializeResp(data []byte) (httpResponse, error) {
	var resp httpResponse

	var yamlResp YAMLResponse
	err := yaml.Unmarshal(data, &yamlResp)
	if err != nil {
		return resp, err
	}

	resp.Headers = yamlResp.Headers
	resp.Body = []byte(yamlResp.Body)
	resp.StatusCode = yamlResp.StatusCode
	return resp, nil
}

// newInteraction creates a new interaction interface with Request and Response that can be encoded in YAML
func (s YAMLSerializer) newInteraction(typeOfInteraction interactionType, req httpRequest, res httpResponse) interaction {
	sReq, _ := s.serializeReq(req)
	var yamlReq YAMLRequest
	yaml.Unmarshal(sReq, &yamlReq)

	sRes, _ := s.serializeResp(res)
	var yamlRes YAMLResponse
	yaml.Unmarshal(sRes, &yamlRes)

	return interaction{
		Type:     typeOfInteraction,
		Request:  yamlReq,
		Response: yamlRes,
	}
}

// encodeCassetteToBytes takes in a cassette and return an []byte of the YAML-encoded cassette
// string() can be called on the []byte for the actual human-readable YAML
func (s YAMLSerializer) encodeCassetteToBytes(cassette cassette) ([]byte, error) {
	return yaml.Marshal(cassette)
}
