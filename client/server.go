package client

type ServerFileInfo struct {
	Files []*FileInfo `json:"files"`
}

type FileInfo struct {
	Path       string `json:"path"`
	Hash       string `json:"hash"`
	Type       int8   `json:"type"`
	SyncStatus int8
}

const (
	TypeFile int8 = 1
	TypeDir  int8 = 2
)

const (
	SyncStatusWait     int8 = 10
	SyncStatusHandling int8 = 20
	SyncStatusFinished int8 = 30
)
