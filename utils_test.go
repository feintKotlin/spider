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
func TestLoadResourceByType(t *testing.T) {
	tags, attrs := loadResourceByType("ij")
	if tags[0] == "img" && tags[1] == "script" &&
		attrs[0] == "src" && attrs[1] == "src" {

	} else {
		t.Error("Wrong File Type:", tags, attrs)
	}
}
