package main

import (
	"io/ioutil"
	"log"
	"net/url"
	"strings"
)

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
