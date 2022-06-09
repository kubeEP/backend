package response

type GCPCluster struct {
	Cluster
	Location string `json:"location"`
}

type GCPDatacenterClusters struct {
	Clusters              []GCPCluster `json:"clusters"`
	IsTemporaryDatacenter bool         `json:"is_temporary_datacenter"`
}
