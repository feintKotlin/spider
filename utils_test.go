package main

import "testing"

func TestUrlHost(t *testing.T) {
	host, err := urlHost("http://www.baidu.com/123")
	if err != nil {
		t.Error("This is correct Url")
	} else {
		if host != "www.baidu.com" {
			t.Error("Wrong Host Name:", host)
		}
	}

}
