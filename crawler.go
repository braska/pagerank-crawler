package crawler

import (
	"net/http"
	"fmt"
	"golang.org/x/net/html"
	"net/url"
	"strings"
	"errors"
	"mime"
	"math"
	"strconv"
)

type visit struct {
	rawUrl string
	requestUrl string
	foundUrls []string
	indexInArray int
}

type job struct {
	refererUrl string
	rawUrl string
}

type Crawler struct {
	Options *Options
	visitedUrls []string
	linksOnPages []int

	matrix map[string]map[string]int
	visits map[string]visit

	queue []job
}

func NewCrawler(opts *Options) *Crawler {
	ret := new(Crawler)
	ret.Options = opts

	ret.visits = make(map[string]visit)
	ret.matrix = make(map[string]map[string]int)
	ret.visitedUrls = []string{}
	ret.linksOnPages = []int{}

	return ret
}

func (c *Crawler) addToMatrix(refererUrl string, requestUrl string) {
	if c.matrix[refererUrl] == nil {
		c.matrix[refererUrl] = make(map[string]int)
	}
	c.matrix[refererUrl][requestUrl] += 1
	c.linksOnPages[c.visits[refererUrl].indexInArray] += 1
}

func (c *Crawler) Run(entryUrl string) {
	v, err := c.visitUrl(entryUrl)

	if err != nil {
		return
	}

	for _, u := range v.foundUrls {
		c.queue = append(c.queue, job{v.requestUrl, u})
	}

	for len(c.queue) != 0 {
		q := c.queue[0]
		c.queue = c.queue[1:]

		if prevVisit, ok := c.visits[q.rawUrl]; ok {
			c.addToMatrix(q.refererUrl, prevVisit.requestUrl);
			continue
		}
		v, err := c.visitUrl(q.rawUrl)

		if err != nil {
			fmt.Printf("%v (Referer: %v)\n", err, q.refererUrl)
			continue
		}

		if prevVisit, ok := c.visits[v.requestUrl]; !ok {
			v.indexInArray = len(c.visitedUrls)
			c.visits[v.rawUrl] = v
			c.visits[v.requestUrl] = v
			c.visitedUrls = append(c.visitedUrls, v.requestUrl)
			c.linksOnPages = append(c.linksOnPages, 0)
			fmt.Println("New link:", v.requestUrl)
		} else {
			c.visits[v.rawUrl] = prevVisit
		}

		c.addToMatrix(q.refererUrl, v.requestUrl)

		for _, u := range v.foundUrls {
			c.queue = append(c.queue, job{v.requestUrl, u})
		}
	}

	p := c.evaluatePagerank()

	sum := 0.0
	for i, pagerank := range p {
		sum += pagerank
		fmt.Println(c.visitedUrls[i], pagerank)
	}

	fmt.Println(sum)
}

func getHref(t html.Token) (ok bool, href string) {
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}

	return
}

func (c *Crawler) isSameHost(u *url.URL, aUrl string) bool {
	au, err := url.Parse(aUrl)

	if err != nil {
		return false
	}

	return au.Host == u.Host
}

func isHtml(res *http.Response) bool {
	contentType := res.Header.Get("Content-type")

	for _, v := range strings.Split(contentType, ",") {
		t, _, err := mime.ParseMediaType(v)
		if err != nil {
			break
		}
		if t == "text/html" {
			return true
		}
	}
	return false
}

func (c *Crawler) visitUrl(urlString string) (visit, error) {
	res, err := http.Head(urlString)

	v := visit{
		requestUrl: res.Request.URL.String(),
		rawUrl: urlString,
		foundUrls: []string{},
	}

	if !isHtml(res) {
		fmt.Println(v.requestUrl, "is not HTML page")
		return v, nil
	}

	if err != nil {
		return v, errors.New("ERROR: Failed to crawl \"" + urlString + "\"");
	}

	res, err = http.Get(urlString)

	if err != nil {
		return v, errors.New("ERROR: Failed to crawl \"" + urlString + "\"");
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 400 {
		return v, errors.New("ERROR: Failed to crawl \"" + urlString + "\" (code " + strconv.Itoa(res.StatusCode) + ")")
	}

	z := html.NewTokenizer(res.Body)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			return v, nil
		case tt == html.StartTagToken:
			t := z.Token()

			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}

			ok, aUrl := getHref(t)
			if !ok {
				continue
			}

			au, err := url.Parse(strings.Trim(aUrl, " "))

			if err != nil {
				continue
			}


			if au.Scheme == "" {
				au.Scheme = res.Request.URL.Scheme
			}

			if au.Host == "" {
				au.Host = res.Request.URL.Host
			}

			if !strings.HasPrefix(au.Path, "/") {
				au.Path = res.Request.URL.Path + au.Path
			}

			au.Fragment = ""

			aUrl = au.String()

			if (au.Scheme != "http" && au.Scheme != "https") || (c.Options.SameHostOnly && !c.isSameHost(res.Request.URL, aUrl)) {
				continue
			}

			v.foundUrls = append(v.foundUrls, aUrl)
		}
	}
}

func (c *Crawler) pagerankIterate(p []float64) []float64 {
	size := len(p)
	new_p := make([]float64, size)

	for j := 0; j < size; j++ {
		new_p[j] = 0
		for i := 0; i < size; i++ {
			probabilityOfClickingOnLink := float64(c.matrix[c.visitedUrls[i]][c.visitedUrls[j]]) / float64(c.linksOnPages[i])
			new_p[j] += probabilityOfClickingOnLink * p[i]
		}
	}

	norm := 0.0
	for j := 0; j < size; j++ {
		norm += new_p[j]
	}

	antinorm := 1.0 / norm

	for j := 0; j < size; j++ {
		p[j] = new_p[j] * antinorm
	}

	return new_p
}

func calculateChange(p, new_p []float64) float64 {
	acc := 0.0

	for i, pForI := range p {
		acc += math.Abs(pForI - new_p[i])
	}

	return acc
}

func (c *Crawler) evaluatePagerank() []float64 {
	size := len(c.visitedUrls)
	inverseOfSize := 1.0 / float64(size)

	p := make([]float64, size)
	for i := range p {
		p[i] = inverseOfSize
	}

	change := 2.0

	for change > 0.0001 {
		new_p := c.pagerankIterate(p)
		change = calculateChange(p, new_p)
		p = new_p
	}

	return p
}
