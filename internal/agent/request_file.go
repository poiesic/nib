package agent

import (
	"encoding/json"
	"fmt"
	"os"
)

// RequestFileEnv is the environment variable used to pass the request file path
// to agent binaries during interactive dispatch.
const RequestFileEnv = "NIB_AGENT_REQUEST_FILE"

// WriteRequestFile writes a JSON-encoded request to a temporary file and
// returns its path. The caller is responsible for removing the file.
func WriteRequestFile(req Request) (string, error) {
	f, err := os.CreateTemp("", "nib-req-*.json")
	if err != nil {
		return "", fmt.Errorf("creating request file: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(req); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("writing request file: %w", err)
	}
	return f.Name(), nil
}

// ReadRequestFile reads and decodes a Request from the given file path,
// then removes the file.
func ReadRequestFile(path string) (Request, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Request{}, fmt.Errorf("reading request file: %w", err)
	}
	os.Remove(path)

	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return Request{}, fmt.Errorf("parsing request file: %w", err)
	}
	return req, nil
}
