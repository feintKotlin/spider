package main

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {
	content, code := Get("http://www.baidu.com")
	if code != 500 {
		fmt.Println(content)
	}
}
