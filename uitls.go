package utils

import (
	"bufio"
	"io"
	"os"
)

//ReadString 读取文本
func ReadString(r io.Reader) (string, error) {
	breader := bufio.NewReader(r)
	bytes, err := breader.ReadBytes(0)
	if err != nil {
		return "", err
	}
	return string(bytes[:len(bytes)-1]), nil
}

//PathExist 文件是否存在
func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}
