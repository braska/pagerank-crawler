package main

import (
	"crawler"
)

func main() {
	opts := crawler.NewOptions()
	opts.SameHostOnly = true
	c := crawler.NewCrawler(opts)

	c.Run("http://www.live-notes.ru/")
}
