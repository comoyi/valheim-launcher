package client

import (
	"github.com/comoyi/valheim-launcher/config"
	"github.com/comoyi/valheim-launcher/log"
	"github.com/comoyi/valheim-launcher/util/cryptoutil/md5util"
	"github.com/comoyi/valheim-launcher/util/fsutil"
	"os"
	"path/filepath"
)

type CacheInfo struct {
	GenerateTime int          `json:"generate_time"`
	Files        []*CacheFile `json:"files"`
}

type CacheFile struct {
	RelativePath string   `json:"relative_path"`
	Type         FileType `json:"type"`
	Hash         string   `json:"hash"`
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
