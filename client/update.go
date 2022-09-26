package client

import (
	"context"
	"github.com/comoyi/valheim-launcher/log"
	"strconv"
	"time"
)

type UpdateInfo struct {
	Current int
	Total   int
}

func (u *UpdateInfo) GetRatio() float64 {
	return float64(u.Current) / float64(u.Total)
}

var UpdateInf *UpdateInfo

func init() {
	UpdateInf = &UpdateInfo{}
}

func update(ctx context.Context, baseDir string, progressChan chan<- struct{}) {
	log.Infof("baseDir: %v\n", baseDir)

	if baseDir == "" {
		log.Warnf("未选择文件夹\n")
		return
	}

	serverFileInfo := &ServerFileInfo{Files: make([]*FileInfo, 0)}

	// test
	for i := 0; i < 10; i++ {
		serverFileInfo.Files = append(serverFileInfo.Files, &FileInfo{
			Name:       "file-name-" + strconv.Itoa(i+1),
			Hash:       "hash-" + strconv.Itoa(i+1),
			SyncStatus: SyncStatusWait,
		})
	}

	serverFiles := serverFileInfo.Files
	fileCount := len(serverFiles)
	log.Debugf("file count %v\n", fileCount)

	UpdateInf.Total = fileCount
	UpdateInf.Current = 0

	var syncChan = make(chan *FileInfo, fileCount)
	for _, file := range serverFiles {
		syncChan <- file
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			select {
			case f := <-syncChan:
				syncFile(f)
				UpdateInf.Current += 1
				go func() {
					progressChan <- struct{}{}
				}()
			default:
				return
			}
		}
	}
}

func syncFile(fileInfo *FileInfo) {
	log.Debugf("syncing file info %+v\n", fileInfo)
	<-time.After(100 * time.Millisecond)
}
