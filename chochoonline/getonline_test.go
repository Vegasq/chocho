package chochoonline

import (
	"golang.org/x/net/html"
	"sort"
	"strings"
	"testing"
)

const paginationHtml = `<html><head></head><body>
<ul class="paging">
<li><a href="#" class="prev">prev</a></li>
<li class="active"><a class="endless_page_link" href="/">1</a></li>
<li><a class="endless_page_link" href="/?page=2">2</a></li>
<li><a class="endless_page_link" href="/?page=3">3</a></li>
<li><a class="endless_page_link" href="/?page=4">4</a></li>
<span class="endless_separator">...</span>
<li><a class="endless_page_link" href="/?page=42">42</a></li>
<li><a href="/?page=2" class="next endless_page_link">next</a></li>
</ul>
</body></html>`

const titlesHtml = `<body>
<div><div class="title"><a href="/test1/">test1t</div></div>
<div><div class="title"><a href="/test2/">test2t</div></div>
<div><div class="title"><a href="/test3/">test3t</div></div>
</body>`

const combinedHtml = `<html><head></head><body>
<div><div class="title"><a href="/test1/">test1t</div></div>
<div><div class="title"><a href="/test2/">test2t</div></div>
<div><div class="title"><a href="/test3/">test3t</div></div>

<ul class="paging">
<li><a href="#" class="prev">prev</a></li>
<li class="active"><a class="endless_page_link" href="/">1</a></li>
<li><a class="endless_page_link" href="/?page=2">2</a></li>
<li><a class="endless_page_link" href="/?page=3">3</a></li>
<li><a class="endless_page_link" href="/?page=4">4</a></li>
<span class="endless_separator">...</span>
<li><a class="endless_page_link" href="/?page=42">42</a></li>
<li><a href="/?page=2" class="next endless_page_link">next</a></li>
</ul>
</body></html>`

func Test_nodeHasClass(t *testing.T) {
	type args struct {
		n   *html.Node
		cls string
	}

	attrClass := html.Attribute{
		Key: "class",
		Val: "findme",
	}
	attrHref := html.Attribute{
		Key: "href",
		Val: "findme",
	}

	nodeTwoAttrs := html.Node{Attr: []html.Attribute{attrClass, attrHref}}
	nodeOnlyHref := html.Node{Attr: []html.Attribute{attrHref}}
	nodeOnlyClass := html.Node{Attr: []html.Attribute{attrClass}}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Have this class among few", args{&nodeTwoAttrs, "findme"}, true},
		{"Have this class", args{&nodeOnlyClass, "findme"}, true},
		{"No such class", args{&nodeOnlyHref, "findme"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nodeHasClass(tt.args.n, tt.args.cls); got != tt.want {
				t.Errorf("nodeHasClass() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAttrByKey(t *testing.T) {
	type args struct {
		n   *html.Node
		key string
	}

	attrClass := html.Attribute{
		Key: "class",
		Val: "findme",
	}
	attrHref := html.Attribute{
		Key: "href",
		Val: "findme",
	}

	nodeTwoAttrs := html.Node{Attr: []html.Attribute{attrClass, attrHref}}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"Get key", args{&nodeTwoAttrs, "class"}, "findme", false},
		{"Get error", args{&nodeTwoAttrs, "klass"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAttrByKey(tt.args.n, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAttrByKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getAttrByKey() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func findUl(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "ul" && nodeHasClass(n, "paging") {
		return n
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result := findUl(c)
		if result.Type == html.ElementNode && result.Data == "ul" && nodeHasClass(result, "paging") {
			return result
		}
	}

	return &html.Node{}
}



func Test_extractPageFromPagination(t *testing.T) {
	type args struct {
		n *html.Node
	}

	paginNode, _ := html.Parse(strings.NewReader(paginationHtml))
	nonPaginNode, _ := html.Parse(strings.NewReader("<div></div>"))
	paginNode = findUl(paginNode)

	tests := []struct {
		name string
		args args
		want int
	}{
		{"Find last page", args{paginNode}, 42},
		{"Last page not found", args{nonPaginNode}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractPageFromPagination(tt.args.n); got != tt.want {
				t.Errorf("extractPageFromPagination() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_paginationSearch(t *testing.T) {
	type args struct {
		n *html.Node
	}
	paginNode, _ := html.Parse(strings.NewReader(paginationHtml))
	nonPaginNode, _ := html.Parse(strings.NewReader("<div></div>"))
	tests := []struct {
		name string
		args args
		want int
	}{
		{"Find last page", args{paginNode}, 42},
		{"Last page not found", args{nonPaginNode}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := paginationSearch(tt.args.n); got != tt.want {
				t.Errorf("paginationSearch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nameSearch(t *testing.T) {
	type args struct {
		nc    chan string
		token *html.Tokenizer
	}

	token1 := html.NewTokenizer(strings.NewReader(titlesHtml))
	token2 := html.NewTokenizer(strings.NewReader("<div></div>"))
	c := make(chan string)

	tests := []struct {
		name string
		args args
		want []string
	}{
		{"Find titles", args{c, token1}, []string{"test1", "test2", "test3"}},
		{"No titles found", args{c, token2}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go nameSearch(tt.args.nc, tt.args.token)

			var bin []string
			for range tt.want {
				name := <- tt.args.nc
				bin = append(bin, name)
			}

			if len(bin) != len(tt.want) {
				t.Errorf("nameSearch() = %v, want %v", bin, tt.want)
			}

			sort.Strings(bin)

			for i := range tt.want {
				if tt.want[i] != bin[i] {
					t.Errorf("nameSearch() = %v, want %v", bin, tt.want)
				}
			}
		})
	}
}

func Test_getNames(t *testing.T) {
	type args struct {
		o          *onlineUsers
		downloader Downloader
		cat        string
		firstPage  int
		lastPage   int
	}

	downloader := func(url string) string {
		return combinedHtml
	}

	ou := onlineUsers{}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{"Collect names", args{&ou, downloader, "", 1, 1}, []string{"test1", "test2", "test3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getNames(tt.args.o, tt.args.downloader, tt.args.cat, tt.args.firstPage, tt.args.lastPage)

			for i := range tt.want {
				if tt.want[i] != tt.args.o.Names[i] {
					t.Errorf("nameSearch() = %v, want %v", tt.args.o.Names, tt.want)
				}
			}
		})
	}
}