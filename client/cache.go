package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/comoyi/valheim-launcher/config"
	"github.com/comoyi/valheim-launcher/log"
	"github.com/comoyi/valheim-launcher/util/cryptoutil/md5util"
	"github.com/comoyi/valheim-launcher/util/fsutil"
	"github.com/comoyi/valheim-launcher/util/timeutil"
	"io"
	"os"
	"path/filepath"
	"time"
)

type CacheInfo struct {
	GenerateTimestamp int64                 `json:"generate_timestamp"`
	GenerateTime      string                `json:"generate_time"`
	UpdateTimestamp   int64                 `json:"update_timestamp"`
	UpdateTime        string                `json:"update_time"`
	Files             map[string]*CacheFile `json:"files"`
}

func NewCacheInfo() *CacheInfo {
	return &CacheInfo{
		GenerateTimestamp: 0,
		GenerateTime:      "",
		UpdateTimestamp:   0,
		UpdateTime:        "",
		Files:             make(map[string]*CacheFile),
	}
}

type CacheFile struct {
	RelativePath string   `json:"relative_path"`
	Type         FileType `json:"type"`
	Hash         string   `json:"hash"`
}

func isRegenerateCacheDb() bool {
	cacheInfoFilePath, err := getCacheInfoFilePath()
	if err != nil {
		return true
	}
	isExist, err := fsutil.LExists(cacheInfoFilePath)
	if err != nil {
		return true
	}
	if isExist {
		return false
	}

	return true
}

func generateCacheDb() error {
	cacheDirPath, err := getCacheDirPath()
	if err != nil {
		return err
	}
	err = os.MkdirAll(cacheDirPath, os.ModePerm)
	if err != nil {
		log.Debugf("create cache dir failed, dir: %s, err: %v\n", cacheDirPath, err)
		return err
	}
	cacheFileInfo, err := getClientFileInfo(cacheDirPath)
	if err != nil {
		log.Warnf("get CacheFileInfo failed, err: %v\n", err)
		return err
	}

	nowTimestamp := time.Now().Unix()
	nowDateTime := timeutil.TimestampToDateTime(nowTimestamp)

	var cacheFiles = make(map[string]*CacheFile, 2000)

	files := cacheFileInfo.Files
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

	return writeCacheDb(cacheInfo)
}

func writeCacheDb(cacheInfo *CacheInfo) error {
	if cacheInfo == nil {
		log.Debugf("writeCacheDb failed, err: cacheInfo is nil\n")
		return fmt.Errorf("cacheInfo is nil")
	}
	j, err := json.Marshal(cacheInfo)
	if err != nil {
		log.Debugf("json encode failed, err: %v, cacheInfo: %+v\n", err, cacheInfo)
		return err
	}
	cacheInfoData := string(j)

	log.Debugf("cache info json: %v\n", cacheInfoData)

	cacheInfoFilePath, err := getCacheInfoFilePath()
	if err != nil {
		log.Debugf("get CacheInfoFilePath failed, err: %v\n", err)
		return err
	}

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

func addCacheDbData(hashSum string, cacheFile *CacheFile) (*CacheInfo, error) {
	cacheInfo, err := getCacheInfo()
	if err != nil {
		return nil, err
	}

	if cacheFile != nil {
		nowTimestamp := time.Now().Unix()
		nowDateTime := timeutil.TimestampToDateTime(nowTimestamp)
		cacheInfo.UpdateTimestamp = nowTimestamp
		cacheInfo.UpdateTime = nowDateTime
		cacheInfo.Files[hashSum] = cacheFile
		err = writeCacheDb(cacheInfo)
		if err != nil {
			return nil, err
		}
	}
	return cacheInfo, nil
}

func getCacheInfoFilePath() (string, error) {

	cacheInfoFileName := "valheim-launcher-cache"
	cacheDir := config.Conf.CacheDir

	cacheDirPath, err := filepath.Abs(cacheDir)
	if err != nil {
		log.Debugf("get cache dir absolute path failed, cache dir: %s, err: %v\n", cacheDir, err)
		return "", err
	}

	err = os.MkdirAll(cacheDirPath, os.ModePerm)
	if err != nil {
		log.Debugf("create cache dir failed, dir: %s, err: %v\n", cacheDirPath, err)
		return "", err
	}

	cacheInfoFilePath := filepath.Join(cacheDirPath, cacheInfoFileName)
	log.Debugf("cache dir: %s, cacheInfoFilePath: %s\n", cacheDir, cacheInfoFilePath)
	return cacheInfoFilePath, nil
}

func getCacheInfo() (*CacheInfo, error) {
	cacheInfoFilePath, err := getCacheInfoFilePath()
	if err != nil {
		log.Debugf("get CacheInfoFilePath failed, err: %v\n", err)
		return nil, err
	}

	isExist, err := fsutil.LExists(cacheInfoFilePath)
	if err != nil {
		return nil, err
	}
	var cacheInfo *CacheInfo = NewCacheInfo()
	if isExist {
		fileContentByte, err := os.ReadFile(cacheInfoFilePath)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(fileContentByte, &cacheInfo)
		if err != nil {
			log.Debugf("decode cacheInfoFile failed, err: %v\n", err)
			return nil, err
		}
	}

	log.Debugf("cacheInfoFile: %+v\n", cacheInfo)
	if cacheInfo.Files != nil {
		log.Debugf("cacheInfoFile->files: %+v\n", cacheInfo.Files)
		for mapKey, mapV := range cacheInfo.Files {
			log.Debugf("cacheInfoFile->files->%v: %+v\n", mapKey, mapV)
		}
	}
	return cacheInfo, nil
}

func checkCache(fileInfo *FileInfo, cacheInfo *CacheInfo) (bool, string, error) {
	cachePath := ""
	if fileInfo.Hash == "" {
		return false, cachePath, nil
	}

	if cacheInfo == nil {
		return false, cachePath, nil
	}

	isHitCache, cacheFile := checkHitCache(fileInfo.Hash, cacheInfo)
	if !isHitCache {
		return false, cachePath, nil
	}

	if cacheFile == nil {
		return false, cachePath, fmt.Errorf("cache data error, cacheFile is nil")
	}
	cacheDir := config.Conf.CacheDir
	cachePathWithoutCacheDir := cacheFile.RelativePath
	cachePath = filepath.Join(cacheDir, cachePathWithoutCacheDir)

	if cachePath == "" {
		log.Debugf("cachePath is empty")
		return false, cachePath, fmt.Errorf("cachePath is empty")
	}
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

func checkHitCache(hashSum string, cacheInfo *CacheInfo) (bool, *CacheFile) {
	if cacheInfo == nil {
		return false, nil
	}

	cacheFile, ok := cacheInfo.Files[hashSum]
	if ok {
		return true, cacheFile
	}
	return false, nil
}

func tryGenerateCacheFile(localPath string, hashSum string, fileType FileType, cacheInfo *CacheInfo) (*CacheInfo, error) {
	isHit, _ := checkHitCache(hashSum, cacheInfo)

	if isHit {
		return nil, nil
	}
	cacheFile, err := generateCacheFile(localPath, hashSum, fileType)
	if err != nil {
		return nil, err
	}

	cacheInfoNew, err := addCacheDbData(hashSum, cacheFile)
	if err != nil {
		return nil, err
	}
	return cacheInfoNew, nil
}

func generateCacheFile(localPath string, hashSum string, fileType FileType) (*CacheFile, error) {
	f, err := os.Open(localPath)
	if err != nil {
		log.Debugf("in generateCacheFile, open localPath file failed, localPath: %s, err: %v\n", localPath, err)
		return nil, err
	}
	defer f.Close()

	cacheDirPath, err := getCacheDirPath()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	nowD := timeutil.TimestampToDate(now.Unix())
	nowT := now.UnixNano()
	cacheFilename := fmt.Sprintf("%s-%v", "vlcache", nowT)
	cacheDirPathT := filepath.Join(cacheDirPath, nowD)
	cacheFilePath := filepath.Join(cacheDirPathT, cacheFilename)

	err = os.MkdirAll(cacheDirPathT, os.ModePerm)
	if err != nil {
		log.Debugf("create cache dir failed, dir: %s, err: %v\n", cacheDirPath, err)
		return nil, err
	}

	file, err := os.Create(cacheFilePath)
	if err != nil {
		log.Debugf("create cache file failed, dir: %s, err: %v\n", cacheDirPathT, err)
		return nil, err
	}
	defer file.Close()

	_, err = io.Copy(file, f)
	if err != nil {
		log.Debugf("write cache file failed, cacheFilePath: %s, err: %v\n", cacheFilePath, err)
		return nil, err
	}

	relativePath, err := filepath.Rel(cacheDirPath, cacheFilePath)
	if err != nil {
		return nil, err
	}

	cacheFile := &CacheFile{
		RelativePath: relativePath,
		Type:         fileType,
		Hash:         hashSum,
	}
	return cacheFile, nil
}

func getCacheDirPath() (string, error) {
	cacheDir := config.Conf.CacheDir

	cacheDirPath, err := filepath.Abs(cacheDir)
	if err != nil {
		log.Debugf("get cache dir absolute path failed, cache dir: %s, err: %v\n", cacheDir, err)
		return "", err
	}

	return cacheDirPath, nil
}
