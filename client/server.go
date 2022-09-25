package client

type ServerFileInfo struct {
	Files []*FileInfo
}

type FileInfo struct {
	Name       string
	Hash       string
	SyncStatus int8
}

const (
	SyncStatusWait     int8 = 10
	SyncStatusHandling int8 = 20
	SyncStatusFinished int8 = 30
)
