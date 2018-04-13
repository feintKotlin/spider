package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//File Type
const (
	IMG = iota
	JS
	CSS
	HTML
)

type file struct {
	name  string
	url   string
	ftype int
}

type page struct {
	content *goquery.Document
	url     string
}

type pageInfo struct {
	url   string
	title string
	intro string
}

type site struct {
	Url string
}

// 错误处理
func handleError(err error, msg string) {
	log.Printf("[ERROR] %s: %s\n", msg, err.Error())
}
func handleErrorFatal(err error, msg string) {
	log.Fatalf("[ERROR] %s: %s\n", msg, err.Error())
}

// 初始化mongodb
func initMongoDB(dbName string, collName string) *mgo.Collection {
	session, err := mgo.Dial("localhost")
	if err != nil {
		handleErrorFatal(err, "Failed to Connect MongoDB")
	}
	c := session.DB(dbName).C(collName)
	return c
}

/**
参数：
	urlCh：从该channel中获取相应的url
	page：将从url请求到的页面内容通过该channel输出
*/
func requestHtml(urlCh <-chan string, pageCh chan<- page, urlMap map[string]bool, done chan bool) {
	client := http.Client{}
	collection := initMongoDB("feint", "site")
	for {
		select {
		case url := <-urlCh:
			result := site{}
			collection.Find(bson.M{"url": url}).One(&result)
			if len(result.Url) == 0 {
				err := collection.Insert(&site{url})
				if err != nil {
					handleErrorFatal(err, "Failed to Insert Row Data")
				}
			} else {
				break
			}
			log.Printf("[URL] Gurrent url [%s]", url)
			req, err := client.Get(url)
			if err != nil {
				handleError(err, "[URL] Failed to request this url")
			} else {
				log.Printf("[URL] Request url [%s] success\n", url)
				doc, err := goquery.NewDocumentFromReader(req.Body)
				if err != nil {
					handleError(err, "[URL] Failed to get response body")
				} else {
					log.Println("[URL] Get Response body success")
					select {
					case pageCh <- page{
						content: doc,
						url:     url,
					}:
					default:
						//log.Println("[PAGE] Page Channel Busy")
					}
				}

				req.Body.Close()
			}
		default:
			//log.Println("================ Spider Finished ================")
			//done <- true
		}
	}
}

func pageContentUtil(fileType int, attr string, url string, selection *goquery.Selection, fileCh chan<- file) {
	var typeStr string
	switch fileType {
	case IMG:
		typeStr = "IMG"
		break
	case JS:
		typeStr = "JS"
		break
	case CSS:
		typeStr = "CSS"
	}

	if src, ok := selection.Attr(attr); ok {
		srcNames := strings.Split(src, "/")
		if absPath, ok := absolutePath(url, src); ok {
			log.Printf("[%s] detect Image tag; src = %s", typeStr, src)
			imgFile := file{
				name:  srcNames[len(srcNames)-1],
				url:   absPath,
				ftype: fileType,
			}
			select {
			case fileCh <- imgFile:
			default:
				//log.Printf("[%s] File Channel Busy\n", typeStr)
			}

		}
	}

}

func dealHtml(pageCh <-chan page, pageInfoCh chan<- pageInfo,
	fileCh chan<- file, urlCh chan<- string, host string) {

	for {
		select {
		case pageCh := <-pageCh:
			// 获取文件资源
			// 图片
			pageCh.content.Find("img").Each(func(i int, selection *goquery.Selection) {
				pageContentUtil(IMG, "src", pageCh.url, selection, fileCh)
			})
			// JS脚本
			pageCh.content.Find("script").Each(func(i int, selection *goquery.Selection) {
				pageContentUtil(JS, "src", pageCh.url, selection, fileCh)
			})
			// CSS样式表
			pageCh.content.Find("link").Each(func(i int, selection *goquery.Selection) {
				if rel, ok := selection.Attr("rel"); ok {
					if rel == "stylesheet" {
						pageContentUtil(CSS, "href", pageCh.url, selection, fileCh)
					}
				}
			})
			// 获取html连接
			pageCh.content.Find("a").Each(func(i int, selection *goquery.Selection) {
				if href, ok := selection.Attr("href"); ok {
					if path, ok := absolutePath(pageCh.url, href); ok {
						log.Printf("[URL] detect Url tag; href = %s\n", href)
						if host != "" {
							if strings.Contains(path, host) {
								select {
								case urlCh <- path:
								default:
									//log.Println("[URL] Url Channel Busy")
								}
							}
						} else {
							select {
							case urlCh <- path:
							default:
								//log.Println("[URL] Url Channel Busy")
							}
						}
					}
				}
			})

			// 获取当前页面的信息
		default:
			//log.Println("[PAGE] Page Channel Busy")
		}
	}
}

func saveFile(fileCh <-chan file, dir string) {
	for {
		select {
		case file := <-fileCh:
			rep, err := http.Get(file.url)
			if err != nil {
				handleError(err, "[FILE] Failed to get file resource")
			} else {
				data, err := ioutil.ReadAll(rep.Body)
				if err != nil {
					handleError(err, "[FILE] Failed to get response body")
				} else {
					fileDir := "image"
					switch file.ftype {
					case CSS:
						fileDir = "css"
						break
					case JS:
						fileDir = "js"
						break
					case IMG:
						fileDir = "image"
						break
					}

					dirCp := strings.Join([]string{dir, fileDir}, "/")
					_, err = os.Stat(dirCp)
					if err != nil {
						if os.IsNotExist(err) {
							os.Mkdir(dirCp, os.ModePerm)
						}
					}
					err = ioutil.WriteFile(strings.Join([]string{dirCp, file.name}, "/"), data, 0666)
					if err != nil {
						handleError(err, "[FILE] Failed to write file")
					} else {
						log.Printf("[FILE] write to file [%s] success\n", file.name)
					}
				}

			}
		default:
			//log.Println("[FILE] File Channel Busy")
		}

	}
}

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

func main() {
	urlCh := make(chan string, 100)
	pageCh := make(chan page, 16)
	fileCh := make(chan file, 24)

	done := make(chan bool)

	startUrl, err := url.Parse("http://www.duowan.com/")
	if err != nil {
		handleError(err, "[URL] Bad Url")
	}

	urlMap := make(map[string]bool)

	go requestHtml(urlCh, pageCh, urlMap, done)
	go dealHtml(pageCh, nil, fileCh, urlCh, "")

	go saveFile(fileCh, "/Users/feint/spider_test")

	urlCh <- startUrl.String()

	<-done
}
