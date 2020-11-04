package playback

import (
	"gopkg.in/yaml.v2"
)

// YAMLSerializer encodes/persists cassettes to files and decodes yaml cassette files.
type YAMLSerializer struct{}

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

func (s YAMLSerializer) serializeResp(input interface{}) ([]byte, error) {
	res := input.(httpResponse)
	yml := YAMLResponse{
		res.Headers,
		string(res.Body),
		res.StatusCode,
	}
	return yaml.Marshal(yml)
}

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

func (s YAMLSerializer) encodeCassetteToBytes(cassette cassette) ([]byte, error) {
	return yaml.Marshal(cassette)
}
