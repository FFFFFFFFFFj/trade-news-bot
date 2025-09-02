package rss

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

//HTTP client with timeout
var client = &http.Client{Timeout: 15 * time.Second}

//---RSS 2.0---
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title string    `xml:"title"`
	Items []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
	DCDate  string `xml:"http://purl.org/dc/elements/1.1/date"`
}

//---Atom---
type Atom struct {
	XMLName xml.Name   `xml:"feed"`
	Entries []AtomItem `xml:"entry"`
}

type AtomItem struct {
	Title   string `xml:"title"`
	Link    Link   `xml:"link"`
	Updated string `xml:"updated"`
}

type Link struct {
	Href string `xml:"href,attr"`
}

//---RSS 1.0 (RDF)---
type RDF struct {
	XMLName xml.Name  `xml:"RDF"`
	Items   []RDFItem `xml:"item"`	
}

type RDFItem struct {
	Title  string `xml:"title"`
	Link   string `xml:"link"`
	DCDate string `xml:"http://purl.org/dc/elements/1.1/date"`
}

// Unified item
type Item struct {
	Title   string
	Link    string
	PubDate string
}

// Fetch loads RSS/Atom/RDF from one sourse
func Fetch(url string) ([]Item, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; TradeNewsBot/1.0)")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	// Try RSS 2.0
	var rss RSS
	if err := xml.Unmarshal(data, &rss); err == nil && len(rss.Channel.Items) > 0 {
		items := make([]Item, 0, len(rss.Channel.Items))
		for _, it := range rss.Channel.Items {
			date := it.PubDate
			if date == "" {
				date = it.DCDate
			}
			if date == "" {
				date = time.Now().Format(time.RFC1123)
			}
			items = append(items, Item{
				Title:   it.Title,
				Link:    it.Link,
				PubDate: date,
			})
		}
		return items, nil
		
	}

	// Try Atom
	var atom Atom
	if err := xml.Unmarshal(data, &atom); err == nil && len(atom.Entries) > 0 {
		items := make([]Item, 0, len(atom.Entries))
		for _, e := range atom.Entries {
			items = append(items, Item{
				Title:   e.Title,
				Link:    e.Link.Href,
				PubDate: e.Updated,
			})
		}
		return items, nil
	}

	// Try RDF RSS 1.0
	var rdf RDF
	if err := xml.Unmarshal(data, &rdf); err == nil && len(rdf.Items) > 0 {
		items := make([]Item, 0, len(rdf.Items))
		for _, it := range rdf.Items {
			date := it.DCDate
			if date == "" {
				date = time.Now().Format(time.RFC1123)
			}
			items = append(items, Item{
				Title:   it.Title,
				Link:    it.Link,
				PubDate: date,
			})
		}
		return items, nil
	}
		
	return nil, fmt.Errorf("unknown feed format from %s", url)
}

//FetchAll loads news from multiple sources
func FetchAll(urls []string) ([]Item, error) {
	var all []Item
	for _, u := range urls {
		items, err := Fetch(u) 
		if err != nil {
			fmt.Println("!!!Source error:", u, err)
			continue // skip source with error
		}
		all = append(all, items...)
	}
	return all, nil
}
