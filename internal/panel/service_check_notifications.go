package panel

import (
	"github.com/kejilion/kejilion-panel/internal/notification"
	"net/http"
)

const serviceCheckNotificationsPath = "/api/v1/cluster/notifications/service-checks"

func (s *Server) handleServiceCheckNotifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet {
		if _, _, ok := s.requireSession(w, r); !ok {
			return
		}
		if s.serviceCheckAlerts == nil {
			s.writeProblem(w, r, 503, "cluster_notifications_unavailable", "Cluster notifications unavailable", "")
			return
		}
		s.writeJSON(w, http.StatusOK, s.serviceCheckAlerts.Snapshot(r.Context()))
		return
	}
	if r.Method != http.MethodPut {
		s.writeProblem(w, r, 405, "method_not_allowed", "Method not allowed", "")
		return
	}
	session, ok := s.requireClusterMutation(w, r)
	if !ok {
		return
	}
	var input notification.CheckAlertInput
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	change := map[string]any{"enabled": input.Enabled, "subscriptions": len(input.Subscriptions)}
	if err := s.audit(r, session.User.ID, "cluster.notifications.service-checks.update", "cluster-notifications", "service-checks", "intent", change); err != nil {
		s.writeProblem(w, r, 503, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	if s.serviceCheckAlerts == nil {
		s.writeProblem(w, r, 503, "cluster_notifications_unavailable", "Cluster notifications unavailable", "")
		return
	}
	result, err := s.serviceCheckAlerts.Configure(r.Context(), input)
	if err != nil {
		_ = s.audit(r, session.User.ID, "cluster.notifications.service-checks.update", "cluster-notifications", "service-checks", "failure", change)
		s.writeNotificationError(w, r, err)
		return
	}
	_ = s.audit(r, session.User.ID, "cluster.notifications.service-checks.update", "cluster-notifications", "service-checks", "success", change)
	s.writeJSON(w, http.StatusOK, result)
}
