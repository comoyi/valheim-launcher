package client

import (
	"bufio"
	"encoding/json"
	"github.com/comoyi/valheim-launcher/config"
	"github.com/comoyi/valheim-launcher/log"
	"github.com/comoyi/valheim-launcher/util/cryptoutil/md5util"
	"github.com/comoyi/valheim-launcher/util/fsutil"
	"github.com/comoyi/valheim-launcher/util/timeutil"
	"os"
	"path/filepath"
	"time"
)

type CacheInfo struct {
	GenerateTimestamp int64                 `json:"generate_timestamp"`
	GenerateTime      string                `json:"generate_time"`
	Files             map[string]*CacheFile `json:"files"`
}

type CacheFile struct {
	RelativePath string   `json:"relative_path"`
	Type         FileType `json:"type"`
	Hash         string   `json:"hash"`
}

// TODO
func isRegenerateCache() bool {
	return true
}

func generateCache(baseDir string) error {

	clientFileInfo, err := getClientFileInfo(baseDir)
	if err != nil {
		log.Warnf("getClientFileInfo failed, err: %v\n", err)
		return err
	}

	nowTimestamp := time.Now().Unix()
	nowDateTime := timeutil.TimestampToDateTime(nowTimestamp)

	var cacheFiles = make(map[string]*CacheFile, 2000)

	files := clientFileInfo.Files
	for _, file := range files {
		cacheFile := &CacheFile{
			RelativePath: file.RelativePath,
			Type:         file.Type,
			Hash:         file.Hash,
		}
		k := file.Hash
		if k != "" {
			cacheFiles[k] = cacheFile
		}
	}

	var cacheInfo *CacheInfo
	cacheInfo = &CacheInfo{
		GenerateTimestamp: nowTimestamp,
		GenerateTime:      nowDateTime,
		Files:             cacheFiles,
	}

	j, err := json.Marshal(cacheInfo)
	if err != nil {
		log.Debugf("json encode failed, err: %v, cacheInfo: %+v\n", err, cacheInfo)
		return err
	}
	cacheInfoData := string(j)

	log.Debugf("cache info json: %v\n", cacheInfoData)

	cacheInfoFileName := "valheim-launcher-cache"
	cacheDir := config.Conf.CacheDir

	cacheDirPath, err := filepath.Abs(cacheDir)
	if err != nil {
		log.Debugf("get cache dir absolute path failed, cache dir: %s, err: %v\n", cacheDir, err)
		return err
	}

	err = os.MkdirAll(cacheDirPath, os.ModePerm)
	if err != nil {
		log.Debugf("create cache dir failed, dir: %s, err: %v\n", cacheDirPath, err)
		return err
	}

	cacheInfoFilePath := filepath.Join(cacheDirPath, cacheInfoFileName)
	log.Debugf("cache dir: %s, cacheInfoFilePath: %s\n", cacheDir, cacheInfoFilePath)

	file, err := os.Create(cacheInfoFilePath)
	if err != nil {
		log.Debugf("create cacheInfoFile failed, err: %v\n", err)
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(cacheInfoData)
	if err != nil {
		log.Debugf("write cacheInfoData failed, err: %v\n", err)
		return err
	}
	err = writer.Flush()
	if err != nil {
		log.Debugf("write cacheInfoData failed[2], err: %v\n", err)
		return err
	}

	return nil
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
