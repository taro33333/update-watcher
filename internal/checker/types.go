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

// RSS2Feed represents an RSS 2.0 feed (used by AWS Security Bulletins, Cloudflare)
type RSS2Feed struct {
	XMLName xml.Name    `xml:"rss"`
	Channel RSS2Channel `xml:"channel"`
}

type RSS2Channel struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
	Items       []RSS2Item `xml:"item"`
}

type RSS2Item struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	PubDate     string   `xml:"pubDate"`
	Description string   `xml:"description"`
	GUID        string   `xml:"guid"`
	Categories  []string `xml:"category"`
}

// NVDResponse represents the NVD API response
type NVDResponse struct {
	ResultsPerPage  int                `json:"resultsPerPage"`
	StartIndex      int                `json:"startIndex"`
	TotalResults    int                `json:"totalResults"`
	Vulnerabilities []NVDVulnerability `json:"vulnerabilities"`
}

type NVDVulnerability struct {
	CVE NVDCVE `json:"cve"`
}

type NVDCVE struct {
	ID           string           `json:"id"`
	Published    string           `json:"published"`
	LastModified string           `json:"lastModified"`
	Descriptions []NVDDescription `json:"descriptions"`
	Metrics      NVDMetrics       `json:"metrics"`
	References   []NVDReference   `json:"references"`
}

type NVDDescription struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type NVDMetrics struct {
	CVSSMetricV31 []NVDCVSSMetric `json:"cvssMetricV31"`
	CVSSMetricV30 []NVDCVSSMetric `json:"cvssMetricV30"`
	CVSSMetricV2  []NVDCVSSMetric `json:"cvssMetricV2"`
}

type NVDCVSSMetric struct {
	Type         string      `json:"type"`
	CVSSData     NVDCVSSData `json:"cvssData"`
	BaseSeverity string      `json:"baseSeverity"`
}

type NVDCVSSData struct {
	Version   string  `json:"version"`
	BaseScore float64 `json:"baseScore"`
}

type NVDReference struct {
	URL string `json:"url"`
}
