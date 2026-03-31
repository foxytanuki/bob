package protocol

const (
	CurrentVersion       = 1
	ActionOpenURL        = "open_url"
	StatusOK             = "OK"
	StatusOpened         = "OPENED"
	StatusInvalidURL     = "INVALID_URL"
	StatusInvalidRequest = "INVALID_REQUEST"
	StatusUnauthorized   = "UNAUTHORIZED"
	StatusDenied         = "DENIED"
	StatusUnreachable    = "UNREACHABLE"
	StatusInternalError  = "INTERNAL_ERROR"
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
	OK      bool   `json:"ok"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type HealthResponse struct {
	OK      bool   `json:"ok"`
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}
