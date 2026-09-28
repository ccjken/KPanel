package panel

import (
	"errors"
	"net/http"
	"strings"

	"github.com/kejilion/kejilion-panel/internal/store"
)

type clusterHostDetailsResponse struct {
	store.ClusterHostDetails
	ResourceVersion string `json:"resourceVersion"`
}

type clusterHostDetailsInput struct {
	store.ClusterHostDetails
	ExpectedResourceVersion string `json:"expectedResourceVersion"`
}

func clusterHostDetailsView(id string, value store.ClusterHostDetails) clusterHostDetailsResponse {
	return clusterHostDetailsResponse{value, store.ClusterHostDetailsResourceVersion(id, value)}
}

func (s *Server) handleClusterHostDetails(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPut {
		s.writeProblem(w, r, http.StatusNotFound, "route_not_found", "Route not found", "")
		return
	}
	session, ok := s.requireClusterMutation(w, r)
	if !ok {
		return
	}
	var input clusterHostDetailsInput
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	input.Price = strings.TrimSpace(input.Price)
	if err := store.ValidateClusterHostDetails(input.ClusterHostDetails); err != nil {
		s.writeProblem(w, r, http.StatusUnprocessableEntity, "cluster_host_details_invalid", "Host details are invalid", "")
		return
	}
	if _, err := s.cluster.Host(r.Context(), id); err != nil {
		s.writeClusterError(w, r, err)
		return
	}
	inventory := s.cluster.Hosts(r.Context())
	ids := make([]string, 0, len(inventory.Items))
	for _, host := range inventory.Items {
		ids = append(ids, host.ID)
	}
	change := map[string]any{"configured": input.ClusterHostDetails != (store.ClusterHostDetails{})}
	if err := s.audit(r, session.User.ID, "cluster.host.details.update", "cluster-host", id, "intent", change); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	if err := s.store.ReplaceClusterHostDetails(id, input.ExpectedResourceVersion, input.ClusterHostDetails, ids); err != nil {
		_ = s.audit(r, session.User.ID, "cluster.host.details.update", "cluster-host", id, "failure", change)
		switch {
		case errors.Is(err, store.ErrConflict):
			s.writeProblem(w, r, http.StatusConflict, "cluster_host_details_changed", "Host details changed", "")
		case errors.Is(err, store.ErrInvalidRecord):
			s.writeProblem(w, r, http.StatusUnprocessableEntity, "cluster_host_details_invalid", "Host details are invalid", "")
		default:
			s.writeProblem(w, r, http.StatusServiceUnavailable, "cluster_host_details_storage_unavailable", "Host details storage unavailable", "")
		}
		return
	}
	s.invalidateClusterShareCache()
	_ = s.audit(r, session.User.ID, "cluster.host.details.update", "cluster-host", id, "success", change)
	s.writeJSON(w, http.StatusOK, clusterHostDetailsView(id, input.ClusterHostDetails))
}
