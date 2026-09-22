package models

type KeyItem struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	TTL  int64  `json:"ttl"` // TTL in seconds, -1 if no expire, -2 if not exists
	Size int64  `json:"size,omitempty"`
}

type ScanKeysResponse struct {
	Keys       []KeyItem `json:"keys"`
	NextCursor uint64    `json:"nextCursor"`
}

type KeyDetailResponse struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	TTL   int64       `json:"ttl"`
	Value interface{} `json:"value"`
}

type CreateKeyInput struct {
	Key        string      `json:"key"`
	Type       string      `json:"type"` // string, hash, list, set, zset, stream
	Value      interface{} `json:"value"`
	TTL        int64       `json:"ttl,omitempty"` // Seconds
	Field      string      `json:"field,omitempty"` // For hash fields or single element additions
	Score      float64     `json:"score,omitempty"` // For zset
}

type RenameKeyInput struct {
	NewKey string `json:"newKey"`
}

type SetTTLInput struct {
	TTL int64 `json:"ttl"` // Seconds. -1 to remove expiration.
}

type ExecuteCommandInput struct {
	Command string `json:"command"`
}

type ExecuteCommandResponse struct {
	Result    string `json:"result"`
	Execution string `json:"execution"`
	Status    string `json:"status"`
}

type PubSubPublishInput struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
}
