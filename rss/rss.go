package rss

import (
	"encoding/xml"
	"net/http"
	"time"
)

//HTTP client with timeout
var client = &http.Client{Timeout: 15 * time.Second}

//Structures for RSS parsing
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title string `xml:"title"`
	Items []Item `xml:"item"`
}

type Item struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
}

//Fetch loads RSS from one source
func Fetch(url string) ([]Item, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rss RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, err
	}
	return rss.Channel.Items, nil
}

//FetchALL loads news from multiple sources
func FetchALL(urls []string) ([]Item, error) {
	var all []Item
	for _, u := range urls {
		items, err := Fetch(u) 
		if err != nil {
			continue // skip source with error
		}
		all = append(all, items...)
	}
	return all, nil
}
