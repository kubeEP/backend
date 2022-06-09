package response

import "github.com/google/uuid"

type GCPDatacenterData struct {
	DatacenterID uuid.UUID `json:"datacenter_id"`
	IsTemporary  bool      `json:"is_temporary"`
}
