package response

import "time"

type NodePoolStatus struct {
	CreatedAt time.Time `json:"created_at"`
	Count     int32     `json:"count"`
}

type HPAStatus struct {
	CreatedAt           time.Time `json:"created_at"`
	Replicas            int32     `json:"replicas"`
	AvailableReplicas   int32     `json:"available_replicas"`
	ReadyReplicas       int32     `json:"ready_replicas"`
	UnavailableReplicas int32     `json:"unavailable_replicas"`
}
