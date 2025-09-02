package rss

import (
	"fmt"
	"github.com/mmcdole/gofeed"
)

// Item - a single news element
type Item struct {
	Title   string
	Link    string
	PubDate string
}

// Fetch downloads and parses  news from the specified RSS/Atom URL
func Fetch(url string) ([]Item, error) {
	parser := gofeed.NewParser()
	feed, err := parser.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}

	items := make([]Item, 0, len(feed.Items))
	for _, i := range feed.Items {
		pubDate := ""
		if i.Published != "" {
			pubDate = i.Published
		} else if i.Updated != "" {
			pubDate = i.Updated
		}

		items = append(items, Item{
			Title:   i.Title,
			Link:    i.Link,
			PubDate: pubDate,
		})
	}
	return items, nil
}

//FetchAll loads news from multiple sources
func FetchAll(urls []string) ([]Item, error) {
	var allItems []Item
	for _, url := range urls {
		items, err := Fetch(url) 
		if err != nil {
			fmt.Println("!!!Source error:", url, err)
			continue // skip source with error
		}
		allItems = append(allItems, items...)
	}
	return allItems, nil
}
