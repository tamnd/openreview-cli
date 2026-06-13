package openreview

import (
	"encoding/json"
	"strings"
)

// Paper is the record emitted for OpenReview notes (papers, preprints).
type Paper struct {
	Rank     int    `json:"rank"`
	ID       string `json:"id"`
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Authors  string `json:"authors"`
	Venue    string `json:"venue"`
	Decision string `json:"decision"`
	Keywords string `json:"keywords"`
	URL      string `json:"url"`
}

// Venue is the record emitted for a known OpenReview conference venue.
type Venue struct {
	Rank int    `json:"rank"`
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// TopVenues is the hardcoded list of prominent ML conference venues.
var TopVenues = []Venue{
	{Rank: 1, ID: "ICLR.cc/2025/Conference", Name: "ICLR 2025", URL: "https://openreview.net/group?id=ICLR.cc/2025/Conference"},
	{Rank: 2, ID: "NeurIPS.cc/2024/Conference", Name: "NeurIPS 2024", URL: "https://openreview.net/group?id=NeurIPS.cc/2024/Conference"},
	{Rank: 3, ID: "ICML.cc/2024/Conference", Name: "ICML 2024", URL: "https://openreview.net/group?id=ICML.cc/2024/Conference"},
	{Rank: 4, ID: "ICLR.cc/2024/Conference", Name: "ICLR 2024", URL: "https://openreview.net/group?id=ICLR.cc/2024/Conference"},
	{Rank: 5, ID: "NeurIPS.cc/2023/Conference", Name: "NeurIPS 2023", URL: "https://openreview.net/group?id=NeurIPS.cc/2023/Conference"},
	{Rank: 6, ID: "ICML.cc/2023/Conference", Name: "ICML 2023", URL: "https://openreview.net/group?id=ICML.cc/2023/Conference"},
	{Rank: 7, ID: "ICLR.cc/2023/Conference", Name: "ICLR 2023", URL: "https://openreview.net/group?id=ICLR.cc/2023/Conference"},
	{Rank: 8, ID: "EMNLP.cc/2024/Conference", Name: "EMNLP 2024", URL: "https://openreview.net/group?id=EMNLP.cc/2024/Conference"},
	{Rank: 9, ID: "ACL.cc/2024/Conference", Name: "ACL 2024", URL: "https://openreview.net/group?id=ACL.cc/2024/Conference"},
	{Rank: 10, ID: "COLM.cc/2024/Conference", Name: "COLM 2024", URL: "https://openreview.net/group?id=COLM.cc/2024/Conference"},
}

// ─── wire types ──────────────────────────────────────────────────────────────

type apiResp struct {
	Notes []wireNote `json:"notes"`
	Count int        `json:"count"`
}

type wireNote struct {
	ID         string          `json:"id"`
	Number     int             `json:"number"`
	Invitation string          `json:"invitation"`
	Content    json.RawMessage `json:"content"`
	CDate      int64           `json:"cdate"`
	MDate      int64           `json:"mdate"`
}

// wireContent decodes the content object of a note. All fields use strVal or
// rawVal to handle both the v2 {"value": ...} wrapper and bare values.
type wireContent struct {
	Title    strVal `json:"title"`
	Abstract strVal `json:"abstract"`
	Authors  rawVal `json:"authors"`
	Keywords rawVal `json:"keywords"`
	PDF      strVal `json:"pdf"`
	Venue    strVal `json:"venue"`
	VenueID  strVal `json:"venueid"`
	Decision strVal `json:"decision"`
}

// strVal handles both {"value": "..."} and a bare "..." string.
type strVal struct {
	Value string
}

func (s *strVal) UnmarshalJSON(b []byte) error {
	var obj struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(b, &obj); err == nil && (obj.Value != "" || isObject(b)) {
		s.Value = obj.Value
		return nil
	}
	return json.Unmarshal(b, &s.Value)
}

// rawVal handles both {"value": [...]} and a bare [...] slice.
type rawVal struct {
	Value []string
}

func (r *rawVal) UnmarshalJSON(b []byte) error {
	var obj struct {
		Value []string `json:"value"`
	}
	if err := json.Unmarshal(b, &obj); err == nil && (obj.Value != nil || isObject(b)) {
		r.Value = obj.Value
		return nil
	}
	return json.Unmarshal(b, &r.Value)
}

// isObject returns true when b is a JSON object (starts with '{').
func isObject(b []byte) bool {
	for _, c := range b {
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		return c == '{'
	}
	return false
}

// wireToPaper converts a wireNote into the exported Paper type.
func wireToPaper(n wireNote, rank int) Paper {
	var c wireContent
	_ = json.Unmarshal(n.Content, &c)

	authors := strings.Join(c.Authors.Value, "; ")

	kws := c.Keywords.Value
	if len(kws) > 3 {
		kws = kws[:3]
	}
	keywords := strings.Join(kws, ", ")

	venue := c.Venue.Value
	if venue == "" {
		venue = c.VenueID.Value
	}

	return Paper{
		Rank:     rank,
		ID:       n.ID,
		Number:   n.Number,
		Title:    c.Title.Value,
		Authors:  authors,
		Venue:    venue,
		Decision: c.Decision.Value,
		Keywords: keywords,
		URL:      "https://openreview.net/forum?id=" + n.ID,
	}
}
