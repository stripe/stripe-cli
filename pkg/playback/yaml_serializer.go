package playback

import (
	"net/http"
	"net/url"

	"gopkg.in/yaml.v2"
)

// YAMLSerializer encodes/persists cassettes to files and decodes yaml cassette files.
type YAMLSerializer struct{}

// EncodeCassette takes in a playback.cassette and returns an []byte of the YAML-encoded cassette
func (s YAMLSerializer) EncodeCassette(cassette Cassette) ([]byte, error) {
	var encodedCassette YAMLCassette
	for _, inter := range cassette {
		req := inter.Request.(httpRequest)
		yamlReq, err := s.serializeReq(req)
		if err != nil {
			return nil, err
		}

		res := inter.Response.(httpResponse)
		yamlRes, err := s.serializeResp(res)
		if err != nil {
			return nil, err
		}

		encodedCassette = append(encodedCassette, YAMLInteraction{Type: inter.Type, Request: yamlReq.(YAMLRequest), Response: yamlRes.(YAMLResponse)})
	}

	return yaml.Marshal(encodedCassette)
}

// DecodeCassette takes in a []byte of YAML and returns a playback.cassette of it
func (s YAMLSerializer) DecodeCassette(data []byte) (Cassette, error) {
	yamlCassette := YAMLCassette{}
	yaml.Unmarshal(data, &yamlCassette)

	var decodedCassette Cassette
	for _, inter := range yamlCassette {
		decodedCassette = append(decodedCassette, interaction{
			Type: inter.Type,
			Request: httpRequest{
				Headers: inter.Request.Headers,
				Body:    []byte(inter.Request.Body),
				Method:  inter.Request.Method,
				URL:     inter.Request.URL,
			},
			Response: httpResponse{
				Headers:    inter.Response.Headers,
				Body:       []byte(inter.Response.Body),
				StatusCode: inter.Response.StatusCode,
			},
		})
	}

	return decodedCassette, nil
}

// -------- SERIALIZATION is used to encode interfaces to YAML
// serializeReq takes in an httpRequest and returns a YAMLRequest
func (s YAMLSerializer) serializeReq(req httpRequest) (interface{}, error) {
	return YAMLRequest{
		req.Method,
		string(req.Body),
		req.Headers,
		req.URL,
	}, nil
}

// serializeResp takes in an httpResponse and returns a YAMLResponse
func (s YAMLSerializer) serializeResp(res httpResponse) (interface{}, error) {
	return YAMLResponse{
		res.Headers,
		string(res.Body),
		res.StatusCode,
	}, nil
}

// -------- DESERIALIZATION is used to decode YAML to interfaces
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

// ----------- The following structs are used internally for conversion to and from YAML

// YAMLRequest is a playback.httpRequest interface encoded to YAML
type YAMLRequest struct {
	Method  string      `yaml:"method"`
	Body    string      `yaml:"body"`
	Headers http.Header `yaml:"headers"`
	URL     url.URL     `yaml:"url"`
}

// YAMLResponse is a playback.httpResponse interface encoded to YAML
type YAMLResponse struct {
	Headers    http.Header `yaml:"headers"`
	Body       string      `yaml:"body"`
	StatusCode int         `yaml:"status_code"`
}

// YAMLInteraction is a playback.interaction interface encoded to YAML
type YAMLInteraction struct {
	Type     interactionType
	Request  YAMLRequest
	Response YAMLResponse
}

// YAMLCassette is a playback.cassette interface encoded to YAML
type YAMLCassette []YAMLInteraction
