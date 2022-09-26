package client

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"github.com/comoyi/valheim-launcher/config"
	"github.com/comoyi/valheim-launcher/log"
	"github.com/comoyi/valheim-launcher/utils/fileutil"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

	resp, err := http.Get(getFullUrl("/files"))
	if err != nil {
		log.Debugf("request failed, err: %v\n", err)
		n := fyne.NewNotification("提示", "从服务器获取文件列表失败")
		fyne.CurrentApp().SendNotification(n)
		return
	}
	j, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Debugf("read file failed, err: %v\n", err)
		return
	}
	var serverFileInfo *ServerFileInfo
	err = json.Unmarshal(j, &serverFileInfo)
	if err != nil {
		log.Debugf("json.Unmarshal failed, err: %v\n", err)
		return
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

syncFile:
	for {
		select {
		case <-ctx.Done():
			return
		default:
			select {
			case f := <-syncChan:
				err := syncFile(f, baseDir)
				if err != nil {
					log.Debugf("sync file failed, fileInfo: %+v, err: %s\n", f, err)
					return
				}
				UpdateInf.Current += 1
				go func() {
					progressChan <- struct{}{}
				}()
			default:
				break syncFile
			}
		}
	}

	deleteFiles(serverFileInfo, baseDir)
}

func syncFile(fileInfo *FileInfo, baseDir string) error {
	var err error
	log.Debugf("syncing file info %+v\n", fileInfo)

	localPathRaw := fmt.Sprintf("%s%s", baseDir, fileInfo.Path)
	localPath := filepath.Clean(localPathRaw)
	log.Debugf("serverPath: %s, localPathRaw: %s, localPath: %s\n", fileInfo.Path, localPathRaw, localPath)

	isExist, err := fileutil.Exists(localPath)
	if err != nil {
		return err
	}

	if fileInfo.Type == TypeDir {
		if isExist {
			return nil
		}
		err = os.MkdirAll(localPath, os.ModePerm)
		if err != nil {
			log.Warnf("os.Mkdir failed, err: %v\n", err)
			return err
		}
	} else {
		if isExist {
			f, err := os.Open(localPath)
			if err != nil {
				return err
			}
			bytes, err := io.ReadAll(f)
			if err != nil {
				return err
			}
			hashSumRaw := md5.Sum(bytes)
			hashSum := fmt.Sprintf("%x", hashSumRaw)
			log.Debugf("file: %s, serverHashSum: %s, hashSum: %s\n", fileInfo.Path, fileInfo.Hash, hashSum)

			if hashSum == fileInfo.Hash {
				log.Debugf("same file skip , localPath: %s\n", localPath)
				return nil
			}
		}
		//log.Debugf("remove local file, localPath: %s\n", localPath)
		//err = os.Remove(localPath)
		//if err != nil {
		//	return err
		//}
		localDir := filepath.Dir(localPath)
		err = os.MkdirAll(localDir, os.ModePerm)
		if err != nil {
			log.Warnf("os.Mkdir failed, err: %v\n", err)
			return err
		}

		file, err := os.Create(localPath)
		if err != nil {
			return err
		}
		defer file.Close()

		q := url.Values{}
		q.Set("file", fileInfo.Path)
		resp, err := http.Get(fmt.Sprintf("%s%s", getFullUrl("/sync"), "?"+q.Encode()))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return err
		}
	}

	log.Debugf("[OK]synced file info %+v\n", fileInfo)

	return nil
}

type ClientFileInfo struct {
	Files []*FileInfo `json:"files"`
}

func deleteFiles(serverFileInfo *ServerFileInfo, baseDir string) {
	clientFileInfo, err := getClientFileInfo(baseDir)
	if err != nil {
		log.Warnf("getClientFileInfo failed, err: %v\n", err)
		return
	}
	files := clientFileInfo.Files
	for _, file := range files {
		if !in(file.Path, serverFileInfo.Files) {

			if !isInAllowDeleteDirs(file.Path) {
				continue
			}

			log.Debugf("will delete, file: %s\n", file.Path)
			path := fmt.Sprintf("%s%s", baseDir, file.Path)
			err := os.RemoveAll(path)
			if err != nil {
				log.Warnf("delete file failed, err: %v, file: %s\n", err, file.Path)
				return
			}
		}
	}
}

func in(file string, files []*FileInfo) bool {
	for _, f := range files {
		if file == f.Path {
			return true
		}
	}
	return false
}

func isInAllowDeleteDirs(file string) bool {
	allowDeleteDirs := make([]string, 0)
	allowDeleteDirs = append(allowDeleteDirs, "BepInEx")
	allowDeleteDirs = append(allowDeleteDirs, "doorstop_libs")
	allowDeleteDirs = append(allowDeleteDirs, "unstripped_corlib")
	for _, f := range allowDeleteDirs {
		if strings.HasPrefix(file, "/"+f) || strings.HasPrefix(file, "\\"+f) {
			return true
		}
	}
	return false
}

func getClientFileInfo(baseDir string) (*ClientFileInfo, error) {
	var clientFileInfo = &ClientFileInfo{}

	files := make([]*FileInfo, 0)

	err := filepath.Walk(baseDir, walkFun(&files, baseDir))
	if err != nil {
		log.Debugf("refresh files info failed\n")
		return nil, err
	}

	clientFileInfo.Files = files
	return clientFileInfo, nil
}

func walkFun(files *[]*FileInfo, baseDir string) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		if !strings.HasPrefix(path, baseDir) {
			log.Warnf("path not excepted, path: %s\n", path)
			return nil
		}
		pathRelative := strings.TrimPrefix(path, baseDir)
		if pathRelative == "" {
			return nil
		}
		var file *FileInfo
		if info.IsDir() {
			file = &FileInfo{
				Path: pathRelative,
				Type: TypeDir,
				Hash: "",
			}
		} else {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			bytes, err := io.ReadAll(f)
			if err != nil {
				return err
			}
			hashSumRaw := md5.Sum(bytes)
			hashSum := fmt.Sprintf("%x", hashSumRaw)
			log.Tracef("file: %s, hashSum: %s\n", path, hashSum)
			file = &FileInfo{
				Path: pathRelative,
				Type: TypeFile,
				Hash: hashSum,
			}
		}
		*files = append(*files, file)
		return nil
	}
}

func getFullUrl(path string) string {
	protocol := config.Conf.Protocol
	if protocol == "" {
		protocol = "http"
	}
	host := config.Conf.Host
	port := config.Conf.Port
	u := fmt.Sprintf("%s://%s:%d%s", protocol, host, port, path)
	return u
}
