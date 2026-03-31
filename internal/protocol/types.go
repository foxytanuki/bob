package protocol

const (
	OpenVersionV1         = 1
	OpenVersionV2         = 2
	CurrentVersion        = OpenVersionV2
	ActionOpenURL         = "open_url"
	StatusOK              = "OK"
	StatusOpened          = "OPENED"
	StatusInvalidURL      = "INVALID_URL"
	StatusInvalidRequest  = "INVALID_REQUEST"
	StatusSessionRequired = "SESSION_REQUIRED"
	StatusSessionNotFound = "SESSION_NOT_FOUND"
	StatusMirrorFailed    = "MIRROR_FAILED"
	StatusUnauthorized    = "UNAUTHORIZED"
	StatusDenied          = "DENIED"
	StatusUnreachable     = "UNREACHABLE"
	StatusInternalError   = "INTERNAL_ERROR"
)

type Source struct {
	App  string `json:"app,omitempty"`
	Host string `json:"host,omitempty"`
	CWD  string `json:"cwd,omitempty"`
}

type OpenRequest struct {
	Version   int    `json:"version,omitempty"`
	Action    string `json:"action,omitempty"`
	URL       string `json:"url"`
	Source    Source `json:"source,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
}

type OpenResponse struct {
	OK            bool   `json:"ok"`
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
	OpenedURL     string `json:"opened_url,omitempty"`
	Rewritten     bool   `json:"rewritten,omitempty"`
	LocalPort     int    `json:"local_port,omitempty"`
	MappingReused bool   `json:"mapping_reused,omitempty"`
}

type OpenRequestV2 struct {
	Version   int    `json:"version,omitempty"`
	Action    string `json:"action,omitempty"`
	Session   string `json:"session"`
	URL       string `json:"url"`
	Source    Source `json:"source,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
}

type HealthResponse struct {
	OK      bool   `json:"ok"`
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}
