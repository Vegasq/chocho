package chochoonline

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type onlineUsers struct {
	Names []string
}

type config struct {
	UrlTemplate string
	Categories  []string
}

type Downloader func (url string) string

func httpGet(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln("Failed to read page", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	return string(data)
}

func extractPageFromPagination(n *html.Node) int {
	counter := 2
	for lc := n.LastChild; lc != nil; lc = lc.PrevSibling {
		if lc.Data == "li" {
			counter--
		}

		if counter == 0 {
			link, err := getAttrByKey(lc.FirstChild, "href")
			if err != nil {
				log.Fatalln("Failed to find pagination", err)
			}

			splitLink := strings.Split(link, "=")

			page, err := strconv.Atoi(splitLink[1])
			if err != nil {
				log.Fatalln("Page not found", err)
			}

			return page
		}
	}
	return 0
}

func getLastPageFromHtml(n *html.Node) int {
	if n.Type == html.ElementNode && n.Data == "ul" && nodeHasClass(n, "paging") {
		return extractPageFromPagination(n)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result := getLastPageFromHtml(c)
		if result > 0 {
			return result
		}
	}

	return 0
}


func getTotalPages(cat string, downloader Downloader, cfg config) int {
	body := downloader(fmt.Sprintf(cfg.UrlTemplate, cat))
	n, err := html.Parse(strings.NewReader(body))

	if err != nil {
		log.Fatalln("Failed to parse body", err)
	}

	return getLastPageFromHtml(n)
}

func nodeHasClass(n *html.Node, cls string) bool {
	var clss []string
	for i := range n.Attr {
		if strings.ToLower(n.Attr[i].Key) == "class" {
			clss = strings.Split(n.Attr[i].Val, " ")

			for i := range clss {
				if clss[i] == cls {
					return true
				}
			}
		}
	}
	return false
}

func getAttrByKey(n *html.Node, key string) (string, error) {
	for i := range n.Attr {
		if n.Attr[i].Key == key {
			return n.Attr[i].Val, nil
		}
	}
	return "", errors.New("key not found")
}


func getAttrByKeyFromToken(token *html.Tokenizer, key string) string {
	k, v, more := token.TagAttr()
	for more = true; more == true;{
		if string(k) == key {
			return string(v)
		}
		k, v, more = token.TagAttr()
	}
	return ""
}

// getTitlesFromTokenizer - collect titles from web page
func getTitlesFromTokenizer(nc chan string, token *html.Tokenizer){
	var lastDivWasTitle bool

	for {
		tt := token.Next()
		if tt == html.ErrorToken {
			break
		}
		name, hasAttr := token.TagName()

		if string(name) == "a" && hasAttr == true {
			href := getAttrByKeyFromToken(token, "href")
			href = strings.Replace(href, "/", "", -1)
			if lastDivWasTitle == true {
				nc <- href
				lastDivWasTitle = false
			}
		} else if string(name) == "div" && hasAttr == true {
			// Search for title
			klass := getAttrByKeyFromToken(token, "class")
			if klass == "title" {
				lastDivWasTitle = true
			} else {
				lastDivWasTitle = false
			}
		}
	}
}

// getNamesFromPage - download single page and pass it to parser
func getNamesFromPage(cat string, cfg config, page int, downloader Downloader, nc chan string, done chan bool){
	body := downloader(fmt.Sprintf(cfg.UrlTemplate, cat, page))
	tok := html.NewTokenizer(strings.NewReader(body))

	getTitlesFromTokenizer(nc, tok)
	done <- true
}

// getNames - spawn goroutines to collect titles from individual pages
func getNames(o *onlineUsers, cfg config, downloader Downloader, cat string, firstPage, lastPage int){
	ch := make(chan string)
	done := make(chan bool)

	for firstPage < lastPage+1 {
		go getNamesFromPage(cat, cfg, firstPage, downloader, ch, done)
		firstPage++
	}

	doneCounter := 0

	// Either exit when we done or in 10 seconds
	start := time.Now()
	for t := time.Now(); t.Sub(start) < time.Second * 10; {
		select {
		case n := <- ch:
			o.Names = append(o.Names, n)
		case <- done:
			doneCounter += 1
			if doneCounter == lastPage {
				logrus.Debug("Collection complete")
				return
			}
		}
	}
}

// onlineByType - split app in 2 different paths.
// 1. Collect information about how many pages are there
// 2. Collect titles from all pages in category
func onlineByType(o *onlineUsers, cat string) {
	pages := getTotalPages(cat, httpGet, getConfig())
	getNames(o, getConfig(), httpGet, cat, 1, pages)
}

func getConfig() config {
	// {"UrlTemplate": "https://site/%s/page/%d", "Categories":  ["pepsi", "fanta"]}
	fl, err := os.Open("config.json")
	if err != nil {
		log.Fatalln("Failed to open config.", err)
	}
	defer fl.Close()

	cfg := config{}
	data, err := ioutil.ReadAll(fl)
	if err != nil {
		log.Fatalln("Failed to read config.", err)
	}

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalln("Failed to unmarshal config", err)
	}

	return cfg
}

func GetOnlineUsers() []string {
	o := onlineUsers{}

	for _, cat := range getConfig().Categories {
		onlineByType(&o, cat)
	}

	return o.Names
}
