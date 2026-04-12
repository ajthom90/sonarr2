package newznab

import (
	"encoding/xml"
	"strconv"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// rssAttr represents a single <newznab:attr name="..." value="..."/> element.
type rssAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// rssEnclosure represents the <enclosure> element inside an <item>.
type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// rssItem represents a single <item> in the RSS channel.
type rssItem struct {
	Title     string       `xml:"title"`
	GUID      string       `xml:"guid"`
	Link      string       `xml:"link"`
	PubDate   string       `xml:"pubDate"`
	Enclosure rssEnclosure `xml:"enclosure"`
	Attrs     []rssAttr    `xml:"http://www.newznab.com/DTD/2010/feeds/attributes/ attr"`
}

// rssChannel is the <channel> element.
type rssChannel struct {
	Items []rssItem `xml:"item"`
}

// rssRoot is the top-level <rss> element.
type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

// parseRss parses a Newznab RSS XML response body into a slice of Releases.
func parseRss(body []byte) ([]indexer.Release, error) {
	var root rssRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, err
	}

	releases := make([]indexer.Release, 0, len(root.Channel.Items))
	for _, item := range root.Channel.Items {
		rel := indexer.Release{
			Title:    item.Title,
			GUID:     item.GUID,
			InfoURL:  item.Link,
			Protocol: indexer.ProtocolUsenet,
		}

		// Download URL comes from the enclosure element.
		if item.Enclosure.URL != "" {
			rel.DownloadURL = item.Enclosure.URL
			if rel.Size == 0 && item.Enclosure.Length > 0 {
				rel.Size = item.Enclosure.Length
			}
		}

		// Parse publish date.
		if item.PubDate != "" {
			if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
				rel.PublishDate = t
			} else if t, err := time.Parse(time.RFC1123, item.PubDate); err == nil {
				rel.PublishDate = t
			}
		}

		// Extract extended attributes from <newznab:attr> elements.
		for _, attr := range item.Attrs {
			switch attr.Name {
			case "size":
				if n, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
					rel.Size = n
				}
			case "category":
				if n, err := strconv.Atoi(attr.Value); err == nil {
					rel.Categories = append(rel.Categories, n)
				}
			}
		}

		releases = append(releases, rel)
	}
	return releases, nil
}
