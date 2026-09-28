package agent

import (
	"github.com/kejilion/kejilion-panel/internal/contract"
	"net/http"
)

func (s *Server) monitoringCheckStatus(w http.ResponseWriter, r *http.Request, requestID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.URL.RawQuery != "" || r.URL.RawPath != "" || r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		writeProblem(w, requestID, http.StatusBadRequest, "invalid_request", "Invalid check status request", "")
		return
	}
	provider, ok := s.monitoring.(interface {
		CheckStatus() contract.ServiceCheckSummary
	})
	if !ok {
		writeProblem(w, requestID, http.StatusServiceUnavailable, "monitoring_unavailable", "Monitoring unavailable", "")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, provider.CheckStatus())
}
