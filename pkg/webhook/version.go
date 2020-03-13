package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/configurator/multitenancy/version"
)

// VersionResponse represents a response to the /version endpoint
type VersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
}

// version returns the current version tag and git commit for the running
// operator binary.
func (s *webhookServer) version(w http.ResponseWriter, r *http.Request) {
	res, err := json.MarshalIndent(VersionResponse{
		Version:   version.Version,
		GitCommit: version.CommitSHA,
	}, "", "  ")
	if err != nil {
		if _, err := w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error()))); err != nil {
			webhooklog.Error(err, "Failed to write response")
		}
		return
	}
	if _, err := w.Write(append(res, []byte("\n")...)); err != nil {
		webhooklog.Error(err, "Failed to write response")
	}
}
