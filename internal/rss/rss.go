package rss

import (
	"encoding/xml"
	"html"
	"io"
	"net/http"
	"regexp"
)

type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

var tagRe = regexp.MustCompile(`(?s)<[^>]*>`)

// stripHTML removes everything that looks like an HTML tag.
func stripHTML(raw string) string {
	withoutTags := tagRe.ReplaceAllString(raw, "")
	return html.UnescapeString(withoutTags)
}

func FetchFeed(feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", `W/"blogo"`)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	feed := &RSSFeed{}
	if unmarshalErr := xml.Unmarshal(body, feed); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	feed.Channel.Title = stripHTML(feed.Channel.Title)
	feed.Channel.Description = stripHTML(feed.Channel.Description)

	for i := range feed.Channel.Items {
		feed.Channel.Items[i].Title = stripHTML(feed.Channel.Items[i].Title)
		feed.Channel.Items[i].Description = stripHTML(feed.Channel.Items[i].Description)
	}

	return feed, nil
}
