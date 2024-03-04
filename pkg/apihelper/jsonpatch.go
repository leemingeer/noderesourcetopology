package apihelper

import (
	"path"
	"strings"
)

// JsonPatch is a json marshaling helper used for patching API objects
type JsonPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value,omitempty"`
}

// NewJsonPatch returns a new JsonPatch object
func NewJsonPatch(verb string, jsonpath string, key string, value string) JsonPatch {
	return JsonPatch{verb, path.Join(jsonpath, strings.ReplaceAll(key, "/", "~1")), value}
}
