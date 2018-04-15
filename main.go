package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
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
	ftype string
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
func requestHtml(urlCh <-chan string, pageCh chan<- page, done chan bool) {
	client := http.Client{}
	collection := initMongoDB("feint", "site")
	collection.Remove(bson.M{})
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

func pageContentUtil(tag string, attr string, url string, selection *goquery.Selection, fileCh chan<- file) {
	if src, ok := selection.Attr(attr); ok {
		srcNames := strings.Split(src, "/")
		if absPath, ok := absolutePath(url, src); ok {
			log.Printf("[%s] detect Image tag; src = %s", strings.ToUpper(tag), src)
			imgFile := file{
				name:  srcNames[len(srcNames)-1],
				url:   absPath,
				ftype: tag,
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
	fileCh chan<- file, urlCh chan<- string, single bool) {
	tags, attrs := loadResourceByType(*restypeOpt)
	for {
		select {
		case pageCh := <-pageCh:
			// 获取文件资源
			log.Println(len(tags), len(attrs))
			for i := 0; i < len(tags); i++ {
				if tags[i] != "stylesheet" {
					pageCh.content.Find(tags[i]).Each(func(index int, selection *goquery.Selection) {
						pageContentUtil(tags[i], attrs[i], pageCh.url, selection, fileCh)
					})
				} else {
					pageCh.content.Find("link").Each(func(index int, selection *goquery.Selection) {
						if rel, ok := selection.Attr("rel"); ok {
							if rel == "stylesheet" {
								pageContentUtil(tags[i], attrs[i], pageCh.url, selection, fileCh)
							}
						}
					})
				}
			}
			// 获取html连接
			pageCh.content.Find("a").Each(func(i int, selection *goquery.Selection) {
				if href, ok := selection.Attr("href"); ok {
					if path, ok := absolutePath(pageCh.url, href); ok {
						log.Printf("[URL] detect Url tag; href = %s\n", href)
						if single {
							host, err := urlHost(pageCh.url)
							if err != nil {
								handleError(err, "Failed to get Host Name")
								return
							}
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
					fileDir := file.ftype

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

var urlfileOpt *string = flag.String("uf", "", "Use -uf <filesource>")
var urlOpt *string = flag.String("u", "https://github.com/feintKotlin/spider", "Use -u <url>")
var singleOpt *bool = flag.Bool("s", true, "Use -s=<true|false>")
var dirOpt *string = flag.String("d", "", "Use -d <filedir>")
var restypeOpt *string = flag.String("t", "i", "Use -t <filetype[i(Image),j(JavaScript),c(CSS),t(txt)]>")

func main() {
	flag.Parse()
	var urlList []string
	// 从命令行的option中获取url列表
	if len(*urlfileOpt) == 0 {
		urlList = append(urlList, *urlOpt)
	} else {
		tempList := urlsFromFile(*urlfileOpt)
		urlList = make([]string, len(tempList))
		copy(urlList, tempList)
	}
	if _, err := os.Stat(*dirOpt); os.IsNotExist(err) {
		handleErrorFatal(err, "Failed to Find Save Directory")
	}
	// 初始化所有将在goroutine中使用到的channel
	urlCh := make(chan string, 100)
	pageCh := make(chan page, 16)
	fileCh := make(chan file, 24)

	done := make(chan bool)

	go requestHtml(urlCh, pageCh, done)
	go dealHtml(pageCh, nil, fileCh, urlCh, *singleOpt)

	go saveFile(fileCh, *dirOpt)

	for _, startUrl := range urlList {
		urlCh <- startUrl
	}

	<-done
}
