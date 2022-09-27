package client

type ServerFileInfo struct {
	Files []*FileInfo `json:"files"`
}

type FileInfo struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Type int8   `json:"type"`
}

const (
	TypeFile int8 = 1
	TypeDir  int8 = 2
)
