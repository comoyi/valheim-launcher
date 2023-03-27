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
	"math/rand"
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

var errServerScanning = fmt.Errorf("服务器正在刷新文件列表，请稍后再试")
var errNotInBaseDir = fmt.Errorf("not in baseDir")

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
			//msg := "服务器正在刷新文件列表，请稍后再试"
			//addMsgWithTime(msg)
			return errServerScanning
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

func syncFile(serverFileInfo *FileInfo, baseDir string) error {
	var err error
	log.Debugf("syncing file info %+v\n", serverFileInfo)

	localPath := filepath.Join(baseDir, serverFileInfo.RelativePath)
	log.Debugf("serverRelativePath: %s, localPath: %s\n", serverFileInfo.RelativePath, localPath)

	isBelong, err := isBelongDir(localPath, baseDir)
	if err != nil {
		return err
	}
	if !isBelong {
		log.Warnf("Not in baseDir, serverRelativePath: %s, localPath: %s, baseDir: %s\n", serverFileInfo.RelativePath, localPath, baseDir)
		return errNotInBaseDir
	}

	isExist, err := fsutil.LExists(localPath)
	if err != nil {
		return err
	}

	syncTypeInfo := ""

	if serverFileInfo.Type == TypeDir {
		if isExist {
			fi, err := os.Lstat(localPath)
			if err != nil {
				return err
			}
			if fi.IsDir() {
				log.Debugf("[SKIP]same dir skip , localPath: %s\n", localPath)
				return nil
			} else {
				log.Debugf("[DELETE]expected a dir but not, delete it, localPath: %s\n", localPath)
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

		syncTypeInfo = "[FROM_LOCAL]"
	} else if serverFileInfo.Type == TypeFile {
		if isExist {
			fi, err := os.Lstat(localPath)
			if err != nil {
				return err
			}
			if fi.Mode().IsRegular() {
				hashSum, err := md5util.SumFile(localPath)
				if err != nil {
					return err
				}
				log.Debugf("file: %s, serverHashSum: %s, hashSum: %s\n", serverFileInfo.RelativePath, serverFileInfo.Hash, hashSum)

				if hashSum == serverFileInfo.Hash {
					log.Debugf("[SKIP]same file skip , localPath: %s\n", localPath)
					return nil
				}
			} else {
				log.Debugf("[DELETE]expected a regular file but not, delete it, localPath: %s\n", localPath)
				err := os.RemoveAll(localPath)
				if err != nil {
					return err
				}
			}
		}

		var srcFile io.ReadCloser
		isCacheHit := false
		cachePath := ""
		if config.Conf.IsUseCache {
			isCacheHit, cachePath, _ = checkCache(serverFileInfo)
		}

		isFinallyUseCache := false
		if isCacheHit {
			srcFile, err = os.Open(cachePath)
			if err != nil {
				return err
			}
			defer srcFile.Close()
			isFinallyUseCache = true

			syncTypeInfo = "[FROM_CACHE]"
		}
		if !isFinallyUseCache {
			resp, err := http.Get(getFullDownloadUrlByFile(serverFileInfo.RelativePath))
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			srcFile = resp.Body
			defer srcFile.Close()

			syncTypeInfo = "[FROM_SERVER]"
		}

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

		_, err = io.Copy(file, srcFile)
		if err != nil {
			return err
		}

		// check hash
		hashSum, err := md5util.SumFile(localPath)
		if err != nil {
			return err
		}
		log.Debugf("check downloaded file hash, file: %s, serverHashSum: %s, hashSum: %s\n", serverFileInfo.RelativePath, serverFileInfo.Hash, hashSum)

		if hashSum != serverFileInfo.Hash {
			return fmt.Errorf("download file hash check failed, expected: %s, got: %s", serverFileInfo.Hash, hashSum)
		}
	} else if serverFileInfo.Type == TypeSymlink {
		resp, err := http.Get(getFullDownloadUrlByFile(serverFileInfo.RelativePath))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		serverLinkDestByte, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		serverLinkDest := string(serverLinkDestByte)

		if isExist {
			fi, err := os.Lstat(localPath)
			if err != nil {
				return err
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				linkDest, err := os.Readlink(localPath)
				if err != nil {
					return err
				}
				if linkDest == serverLinkDest {
					log.Debugf("[SKIP]same symlink skip , localPath: %s\n", localPath)
					return nil
				} else {
					err := os.RemoveAll(localPath)
					if err != nil {
						return err
					}
				}
			} else {
				log.Debugf("[DELETE]expected a symlink but not, delete it, localPath: %s\n", localPath)
				err := os.RemoveAll(localPath)
				if err != nil {
					return err
				}
			}
		}

		localDir := filepath.Dir(localPath)
		err = os.MkdirAll(localDir, os.ModePerm)
		if err != nil {
			return err
		}

		err = os.Symlink(serverLinkDest, localPath)
		if err != nil {
			return err
		}
	}

	log.Debugf("[SYNC]%ssynced info %+v\n", syncTypeInfo, serverFileInfo)

	return nil
}

type ClientFileInfo struct {
	Files []*FileInfo `json:"files"`
}

func deleteFiles(serverFileInfo *ServerFileInfo, baseDir string) error {
	var err error
	clientFileInfo, err := getClientFileInfoWithoutHash(baseDir)
	if err != nil {
		log.Warnf("getClientFileInfo failed, err: %v\n", err)
		return err
	}
	files := clientFileInfo.Files
	for _, file := range files {
		if !in(file.RelativePath, serverFileInfo.Files) {
			path := filepath.Join(baseDir, file.RelativePath)
			isAllow, err := isAllowDelete(path, baseDir, file.RelativePath)
			if err != nil {
				log.Warnf("check delete file failed, err: %v, file: %s\n", err, file.RelativePath)
				return err
			}
			if isAllow {
				err = os.RemoveAll(path)
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

func isAllowDelete(path string, baseDir string, relativePath string) (bool, error) {
	isBelong, err := isBelongDir(path, baseDir)
	if err != nil {
		return false, err
	}
	if !isBelong {
		log.Warnf("Not in baseDir, relativePath: %s, path: %s, baseDir: %s\n", relativePath, path, baseDir)
		return false, nil
	}
	allowDeleteDirs := make([]string, 0)
	allowDeleteDirs = append(allowDeleteDirs, "BepInEx")
	allowDeleteDirs = append(allowDeleteDirs, "doorstop_libs")
	allowDeleteDirs = append(allowDeleteDirs, "unstripped_corlib")
	for _, dir := range allowDeleteDirs {
		if strings.HasPrefix(relativePath, dir) {
			return true, nil
		}
	}
	return false, nil
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
		if path == baseDir {
			return nil
		}
		if !strings.HasPrefix(path, baseDir) {
			log.Warnf("path not expected, baseDir: %s, path: %s\n", baseDir, path)
			return fmt.Errorf("path not expected, baseDir: %s, path: %s\n", baseDir, path)
		}
		relativePath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		if relativePath == "." ||
			relativePath == ".." ||
			strings.HasPrefix(relativePath, "./") ||
			strings.HasPrefix(relativePath, ".\\") ||
			strings.HasPrefix(relativePath, "../") ||
			strings.HasPrefix(relativePath, "..\\") {
			return fmt.Errorf("relativePath not expected, baseDir: %s, path: %s, relativePath: %s\n", baseDir, path, relativePath)
		}
		if relativePath == "" {
			return nil
		}
		var file *FileInfo
		if info.IsDir() {
			log.Tracef("dir:     %s\n", relativePath)
			file = &FileInfo{
				RelativePath: relativePath,
				Type:         TypeDir,
				Hash:         "",
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			log.Tracef("symlink: %s\n", relativePath)
			file = &FileInfo{
				RelativePath: relativePath,
				Type:         TypeSymlink,
				Hash:         "",
			}
		} else if info.Mode().IsRegular() {
			var hashSum string
			if isHash {
				var err error
				hashSum, err = md5util.SumFile(path)
				if err != nil {
					return err
				}
				log.Tracef("file:    %s, hashSum: %s\n", relativePath, hashSum)
			} else {
				log.Tracef("file:    %s\n", relativePath)
			}
			file = &FileInfo{
				RelativePath: relativePath,
				Type:         TypeFile,
				Hash:         hashSum,
			}
		} else {
			log.Tracef("unhandled file type, filepath:  %s\n", relativePath)
			return nil
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

const (
	DownloadServerTypeOss = 2
)

func getFullDownloadUrlByFile(relativePath string) string {
	downloadServers := config.Conf.DownloadServers
	count := len(downloadServers)
	randNum := rand.Intn(count)
	downloadServer := downloadServers[randNum]

	var u string = ""
	prefixPath := downloadServer.PrefixPath
	if downloadServer.Type == DownloadServerTypeOss {
		u = fmt.Sprintf("%s%s", getFullDownloadUrl(downloadServer, fmt.Sprintf("/%s", prefixPath)), relativePath)
	} else {
		q := url.Values{}
		q.Set("file", fmt.Sprintf("%s%s", prefixPath, relativePath))
		u = fmt.Sprintf("%s%s", getFullDownloadUrl(downloadServer, "/sync"), "?"+q.Encode())
	}
	log.Debugf("download from: %s\n", u)
	return u
}

func getFullDownloadUrl(downloadServer *config.DownloadServer, path string) string {
	protocol := downloadServer.Protocol
	if protocol == "" {
		protocol = "http"
	}
	host := downloadServer.Host
	port := downloadServer.Port
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

func checkCache(fileInfo *FileInfo) (bool, string, error) {
	cacheDir := config.Conf.CacheDir
	cachePath := ""

	cacheDirPath, err := filepath.Abs(cacheDir)
	if err != nil {
		log.Debugf("get cache dir absolute path failed, cache dir: %s, err: %v\n", cacheDir, err)
		return false, cachePath, err
	}
	cachePath = filepath.Join(cacheDirPath, fileInfo.RelativePath)
	log.Debugf("cache dir: %s, cache path: %s\n", cacheDir, cachePath)
	isCacheExist, err := fsutil.LExists(cachePath)
	if err != nil {
		log.Debugf("check file is exists failed, cachePath: %s, err: %v\n", cachePath, err)
		return false, cachePath, err
	}
	if isCacheExist {
		cfi, err := os.Lstat(cachePath)
		if err != nil {
			log.Debugf("get file info failed, cachePath: %s, err: %v\n", cachePath, err)
			return false, cachePath, err
		}
		if cfi.Mode().IsRegular() {
			hashSum, err := md5util.SumFile(cachePath)
			if err != nil {
				log.Debugf("get file hash failed, cachePath: %s, err: %v\n", cachePath, err)
				return false, cachePath, err
			}
			log.Debugf("cache path: %s, serverHashSum: %s, cache hashSum: %s\n", cachePath, fileInfo.Hash, hashSum)
			if hashSum == fileInfo.Hash {
				log.Debugf("[CACHE_HIT]cache hit , cachePath: %s\n", cachePath)
				return true, cachePath, nil
			}
		}
	}
	return false, cachePath, nil
}

func isBelongDir(path string, baseDir string) (bool, error) {
	if path == baseDir {
		return true, nil
	}

	// path end with slash
	if strings.HasSuffix(baseDir, "/") ||
		strings.HasSuffix(baseDir, "\\") {
		if strings.HasPrefix(path, baseDir) {
			return true, nil
		}
	} else {
		if strings.HasPrefix(path, baseDir+"/") ||
			strings.HasPrefix(path, baseDir+"\\") {
			return true, nil
		}
	}
	return false, nil
}
