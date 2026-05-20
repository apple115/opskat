package etcd_svc

// EtcdKeyValue represents a single etcd key-value pair.
type EtcdKeyValue struct {
	Key            string `json:"key"`
	Value          string `json:"value"`
	CreateRevision int64  `json:"createRevision"`
	ModRevision    int64  `json:"modRevision"`
	Version        int64  `json:"version"`
	Lease          int64  `json:"lease"`
}

// EtcdRangeRequest controls a range query.
type EtcdRangeRequest struct {
	AssetID int64  `json:"assetId"`
	Prefix  string `json:"prefix"`
	RangeFrom string `json:"rangeFrom,omitempty"`
	RangeTo   string `json:"rangeTo,omitempty"`
	Limit   int64  `json:"limit"`
}

// EtcdRangeResponse holds range results.
type EtcdRangeResponse struct {
	Keys    []EtcdKeyValue `json:"keys"`
	Count   int64          `json:"count"`
	HasMore bool           `json:"hasMore"`
}

// EtcdKeyRequest requests details for a single key.
type EtcdKeyRequest struct {
	AssetID int64  `json:"assetId"`
	Key     string `json:"key"`
}

// EtcdKeyDetail holds metadata for a single key.
type EtcdKeyDetail struct {
	Key            string `json:"key"`
	Value          string `json:"value"`
	CreateRevision int64  `json:"createRevision"`
	ModRevision    int64  `json:"modRevision"`
	Version        int64  `json:"version"`
	Lease          int64  `json:"lease"`
}

// EtcdPutRequest writes a key.
type EtcdPutRequest struct {
	AssetID int64  `json:"assetId"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	LeaseID int64  `json:"leaseId,omitempty"`
}

// EtcdDeleteRequest deletes keys.
type EtcdDeleteRequest struct {
	AssetID int64    `json:"assetId"`
	Keys    []string `json:"keys"`
	Prefix  string   `json:"prefix,omitempty"`
}

// EtcdStatus holds cluster endpoint status.
type EtcdStatus struct {
	Version   string `json:"version"`
	DBSize    int64  `json:"dbSize"`
	Leader    uint64 `json:"leader"`
	RaftIndex uint64 `json:"raftIndex"`
	RaftTerm  uint64 `json:"raftTerm"`
}

// EtcdMember represents a cluster member.
type EtcdMember struct {
	ID         uint64   `json:"id"`
	Name       string   `json:"name"`
	PeerURLs   []string `json:"peerURLs"`
	ClientURLs []string `json:"clientURLs"`
	IsLearner  bool     `json:"isLearner"`
}

// EtcdCommandHistoryEntry tracks executed commands.
type EtcdCommandHistoryEntry struct {
	AssetID     int64  `json:"assetId"`
	Command     string `json:"command"`
	CostMillis  int64  `json:"costMillis"`
	Error       string `json:"error,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}
