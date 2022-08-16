package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"
)

func searchFileGo(waiter *sync.WaitGroup, fileChan chan string, search []byte, buf []byte) {
	defer waiter.Done()
	var ok bool
	var err error
	var file string
	var filestream *os.File
	for {
		if file, ok = <-fileChan; !ok {
			break
		}
		if filestream, err = os.OpenFile(file, os.O_RDONLY, 0644); err != nil {
			if !strings.HasSuffix(err.Error(), "permission denied") {
				log.Printf("打开文件[%s]错误: %v", file, err)
			}
			continue
		}
		var found int
		var readn int
		for {
			if readn, err = filestream.Read(buf); err != nil {
				if err != io.EOF && !strings.HasSuffix(err.Error(), "permission denied") {
					log.Printf("读取文件[%s]错误: %v", file, err)
				}
				break
			}
			found = bytes.Index(buf[:readn], search)
			if found != -1 {
				break
			}
		}
		filestream.Close()
		if found != -1 {
			log.Printf("搜索文件[%s]找到(%#x)", file, found)
		}
	}
}

func searchDir(fileChan chan string, dir string, search []byte) {
	rd, err := os.ReadDir(dir)
	if err != nil {
		if !strings.HasSuffix(err.Error(), "permission denied") {
			log.Printf("读取文件夹[%s]错误: %v", dir, err)
		}
		return
	}
	for _, fi := range rd {
		pathname := path.Join(dir, fi.Name())
		if fi.IsDir() {
			searchDir(fileChan, pathname, search)
		} else if info, err := fi.Info(); err == nil {
			if info.Mode()&os.ModeSymlink == 0 {
				fileChan <- pathname
			}
		}
	}
}

func main() {
	fDir := flag.String("dir", "/", "搜索目录")
	fText := flag.String("text", "", "搜索文本")
	fTextZero := flag.String("textz", "", "搜索文本,包含0结尾")
	fHex := flag.String("hex", "", "搜索二进制")
	fBase64 := flag.String("base64", "", "搜索二进制base64格式")
	flag.Parse()

	if *fDir == "" {
		log.Printf("请设置要搜索的目录")
		return
	}
	if *fText == "" && *fTextZero == "" && *fHex == "" && *fBase64 == "" {
		log.Printf("请设置要搜索的内容")
		return
	}
	var search []byte
	if *fText != "" {
		search = []byte(*fText)
	} else if *fTextZero != "" {
		search = append([]byte(*fTextZero), 0)
	} else if *fHex != "" {
		search, _ = hex.DecodeString(*fHex)
	} else if *fBase64 != "" {
		search, _ = base64.StdEncoding.DecodeString(*fBase64)
	}

	maxThread := 10

	fileChan := make(chan string, maxThread)
	fileWaiter := &sync.WaitGroup{}
	fileWaiter.Add(maxThread)
	for i := 0; i < maxThread; i++ {
		go searchFileGo(fileWaiter, fileChan, search, make([]byte, 4096))
	}
	searchDir(fileChan, *fDir, search)
	close(fileChan)
	fileWaiter.Wait()
}
