package cluster

import "encoding/json"

// Live check observations never enter the legacy cluster store. Keeping the
// persisted snapshot shape unchanged permits old strict decoders on downgrade.
// Notifications consume the in-memory value and persist their own incident state.
func (s HostSnapshot) MarshalJSON() ([]byte, error) {
	type snapshot HostSnapshot
	value := snapshot(s)
	value.Telemetry.ServiceChecks = nil
	return json.Marshal(value)
}
