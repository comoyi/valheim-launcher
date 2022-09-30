package md5util

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

func SumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}
	hashSum := fmt.Sprintf("%x", h.Sum(nil))
	return hashSum, nil
}

func SumString(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}
