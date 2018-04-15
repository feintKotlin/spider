package main

import (
	"io/ioutil"
	"log"
	"net/url"
	"strings"
)

// 从type参数中解析需要获取的文件类型
func loadResourceByType(resType string) (tags []string, attrs []string) {
	for _, t := range resType {
		switch t {
		case 'i':
			tags = append(tags, "img")
			attrs = append(attrs, "src")
		case 'j':
			tags = append(tags, "script")
			attrs = append(attrs, "src")
		case 'c':
			tags = append(tags, "stylesheet")
			attrs = append(attrs, "href")
		}
	}
	return
}

// 获取url的绝对路径
// urlStr：带有hostname的url
// src：输入的url
func absolutePath(urlStr, src string) (string, bool) {
	absPath := src
	if len(src) < 4 {
		return "", false
	}
	if src[0:2] == "//" {
		absPath = "http:" + absPath
	} else if src[0:1] == "/" || src[0:4] != "http" {
		url_, err := url.Parse(urlStr)
		if err != nil {
			handleError(err, "Bad Url")
		} else {
			if url_.Host[len(url_.Host)-1] == '/' {
				url_.Host = url_.Host[:len(url_.Host)]
			}
			if src[0] == '/' {
				src = src[1:]
			}
			absPath = url_.Scheme + "://" + url_.Host + "/" + src
		}

	}
	return absPath, true
}

// 从指定文件中读取初始url列表
func urlsFromFile(filePath string) []string {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		handleErrorFatal(err, "Failed to Read Url File")
		return []string{}
	} else {
		return strings.Split(strings.Trim(string(data), "\n"), "\n")
	}
}

// 错误处理
func handleError(err error, msg string) {
	log.Printf("[ERROR] %s: %s\n", msg, err.Error())
}
func handleErrorFatal(err error, msg string) {
	log.Fatalf("[ERROR] %s: %s\n", msg, err.Error())
}

// 获取url中的host
func urlHost(urlStr string) (host string, err error) {
	startUrl, err := url.Parse(urlStr)
	host = startUrl.Host
	return
}
