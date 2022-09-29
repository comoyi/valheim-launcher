package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/comoyi/valheim-launcher/config"
	"github.com/comoyi/valheim-launcher/log"
	"github.com/comoyi/valheim-launcher/util/cryptoutil/md5util"
	"github.com/comoyi/valheim-launcher/util/fsutil"
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

var UpdateInf *UpdateInfo = &UpdateInfo{}

func update(ctx context.Context, baseDir string, progressChan chan<- struct{}) error {
	log.Infof("baseDir: %v\n", baseDir)

	if baseDir == "" {
		log.Warnf("未选择文件夹\n")
		return fmt.Errorf("invalid base dir")
	}

	j, err := httpGet(getFullUrl("/files"))
	if err != nil {
		log.Debugf("request failed, err: %v\n", err)
		addMsgWithTime("从服务器获取文件列表失败")
		return err
	}
	var serverFileInfo *ServerFileInfo
	err = json.Unmarshal([]byte(j), &serverFileInfo)
	if err != nil {
		log.Debugf("json.Unmarshal failed, err: %v\n", err)
		return err
	}

	scanStatus := serverFileInfo.ScanStatus
	if scanStatus != ScanStatusCompleted {
		if scanStatus == ScanStatusScanning {
			msg := "服务器正在刷新文件列表，请稍后再试"
			addMsgWithTime(msg)
			return fmt.Errorf(msg)
		} else if scanStatus == ScanStatusFailed {
			msg := "服务器刷新文件列表失败"
			addMsgWithTime(msg)
			return fmt.Errorf(msg)
		} else if scanStatus == ScanStatusWait {
			msg := "等待服务器刷新文件列表，请稍后再试"
			addMsgWithTime(msg)
			return fmt.Errorf(msg)
		}
		msg := "服务器异常，请稍后再试"
		addMsgWithTime(msg)
		return fmt.Errorf(msg)
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
			return nil
		default:
			select {
			case f := <-syncChan:
				err := syncFile(f, baseDir)
				if err != nil {
					log.Debugf("sync file failed, fileInfo: %+v, err: %s\n", f, err)
					return err
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

	err = deleteFiles(serverFileInfo, baseDir)
	if err != nil {
		return err
	}

	return nil
}

func syncFile(fileInfo *FileInfo, baseDir string) error {
	var err error
	log.Debugf("syncing file info %+v\n", fileInfo)

	localPath := filepath.Join(baseDir, fileInfo.RelativePath)
	log.Debugf("serverRelativePath: %s, localPath: %s\n", fileInfo.RelativePath, localPath)

	isExist, err := fsutil.Exists(localPath)
	if err != nil {
		return err
	}

	if fileInfo.Type == TypeDir {
		if isExist {
			fi, err := os.Stat(localPath)
			if err != nil {
				return err
			}
			if fi.IsDir() {
				log.Debugf("[SKIP]same dir skip , localPath: %s\n", localPath)
				return nil
			} else {
				log.Debugf("[DELETE]expected a dir but a file, delete it, localPath: %s\n", localPath)
				err := os.RemoveAll(localPath)
				if err != nil {
					return err
				}
			}
		}
		err = os.MkdirAll(localPath, os.ModePerm)
		if err != nil {
			return err
		}
	} else {
		if isExist {
			fi, err := os.Stat(localPath)
			if err != nil {
				return err
			}
			if fi.IsDir() {
				log.Debugf("[DELETE]expected a file but a dir, delete it, localPath: %s\n", localPath)
				err := os.RemoveAll(localPath)
				if err != nil {
					return err
				}
			} else {
				hashSum, err := md5util.SumFile(localPath)
				if err != nil {
					return err
				}
				log.Debugf("file: %s, serverHashSum: %s, hashSum: %s\n", fileInfo.RelativePath, fileInfo.Hash, hashSum)

				if hashSum == fileInfo.Hash {
					log.Debugf("[SKIP]same file skip , localPath: %s\n", localPath)
					return nil
				}
			}
		}

		q := url.Values{}
		q.Set("file", fileInfo.RelativePath)
		resp, err := http.Get(fmt.Sprintf("%s%s", getFullUrl("/sync"), "?"+q.Encode()))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		localDir := filepath.Dir(localPath)
		err = os.MkdirAll(localDir, os.ModePerm)
		if err != nil {
			return err
		}

		file, err := os.Create(localPath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return err
		}
	}

	log.Debugf("[SYNC]synced file info %+v\n", fileInfo)

	return nil
}

type ClientFileInfo struct {
	Files []*FileInfo `json:"files"`
}

func deleteFiles(serverFileInfo *ServerFileInfo, baseDir string) error {
	clientFileInfo, err := getClientFileInfoWithoutHash(baseDir)
	if err != nil {
		log.Warnf("getClientFileInfo failed, err: %v\n", err)
		return err
	}
	files := clientFileInfo.Files
	for _, file := range files {
		if !in(file.RelativePath, serverFileInfo.Files) {
			if isInAllowDeleteDirs(file.RelativePath) {
				path := filepath.Join(baseDir, file.RelativePath)
				err := os.RemoveAll(path)
				if err != nil {
					log.Warnf("delete file failed, err: %v, file: %s\n", err, file.RelativePath)
					return err
				}
				log.Debugf("[DELETE]delete, localPath: %s\n", path)
			}
		}
	}
	return nil
}

func in(file string, files []*FileInfo) bool {
	for _, f := range files {
		if file == filepath.Clean(f.RelativePath) {
			return true
		}
	}
	return false
}

func isInAllowDeleteDirs(relativePath string) bool {
	allowDeleteDirs := make([]string, 0)
	allowDeleteDirs = append(allowDeleteDirs, "BepInEx")
	allowDeleteDirs = append(allowDeleteDirs, "doorstop_libs")
	allowDeleteDirs = append(allowDeleteDirs, "unstripped_corlib")
	for _, dir := range allowDeleteDirs {
		if strings.HasPrefix(relativePath, dir) || strings.HasPrefix(relativePath, "/"+dir) || strings.HasPrefix(relativePath, "\\"+dir) {
			return true
		}
	}
	return false
}

func getClientFileInfoWithoutHash(baseDir string) (*ClientFileInfo, error) {
	return doGetClientFileInfo(baseDir, false)
}

func getClientFileInfo(baseDir string) (*ClientFileInfo, error) {
	return doGetClientFileInfo(baseDir, true)
}

func doGetClientFileInfo(baseDir string, isHash bool) (*ClientFileInfo, error) {
	var clientFileInfo = &ClientFileInfo{}

	files := make([]*FileInfo, 0)

	err := filepath.Walk(baseDir, walkFun(&files, baseDir, isHash))
	if err != nil {
		log.Debugf("refresh files info failed\n")
		return nil, err
	}

	clientFileInfo.Files = files
	return clientFileInfo, nil
}

func walkFun(files *[]*FileInfo, baseDir string, isHash bool) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasPrefix(path, baseDir) {
			log.Warnf("path not expected, baseDir: %s, path: %s\n", baseDir, path)
			return fmt.Errorf("path not expected, baseDir: %s, path: %s\n", baseDir, path)
		}
		relativePath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(relativePath, ".") {
			return fmt.Errorf("relativePath not expected, baseDir: %s, path: %s, relativePath: %s\n", baseDir, path, relativePath)
		}
		if relativePath == "" {
			return nil
		}
		var file *FileInfo
		if info.IsDir() {
			log.Tracef("dir:  %s\n", relativePath)
			file = &FileInfo{
				RelativePath: relativePath,
				Type:         TypeDir,
				Hash:         "",
			}
		} else {
			var hashSum string
			if isHash {
				var err error
				hashSum, err = md5util.SumFile(path)
				if err != nil {
					return err
				}
				log.Tracef("file: %s, hashSum: %s\n", relativePath, hashSum)
			} else {
				log.Tracef("file: %s\n", relativePath)
			}
			file = &FileInfo{
				RelativePath: relativePath,
				Type:         TypeFile,
				Hash:         hashSum,
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

func httpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	j, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(j), nil
}
