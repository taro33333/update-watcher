package checker

import "encoding/xml"

// AtomFeed represents an Atom feed structure (GCP uses Atom format)
type AtomFeed struct {
	Entries []AtomEntry `xml:"entry"`
}

type AtomEntry struct {
	Title     string   `xml:"title"`
	Link      AtomLink `xml:"link"`
	Published string   `xml:"published"`
	Updated   string   `xml:"updated"`
	Summary   string   `xml:"summary"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
}

// GitHubRelease represents a GitHub Release
type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

// GitHubAdvisory represents a GitHub Security Advisory
type GitHubAdvisory struct {
	ID          string `json:"ghsa_id"`
	Summary     string `json:"summary"`
	Severity    string `json:"severity"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

// RSSFeed represents an RSS 1.0 feed (RDF format used by Debian)
type RSSFeed struct {
	XMLName xml.Name   `xml:"RDF"`
	Channel RSSChannel `xml:"channel"`
	Items   []RSSItem  `xml:"item"`
}

type RSSChannel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Date        string `xml:"date"`
	Description string `xml:"description"`
}
