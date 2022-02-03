package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Item struct {
	XMLName struct{} `xml:"item"`

	URL         string     `xml:"-"`
	GUID        string     `xml:"guid,omitempty"`
	Title       string     `xml:"title,omitempty"`
	Description string     `xml:"description,omitempty"`
	PublishDate time.Time  `xml:"pubDate,omitempty"`
	Enclosure   *Enclosure `xml:"enclosure,omitempty"`
	Duration    string     `xml:"itunes:duration,omitempty"`
	Keywords    string     `xml:"itunes:keywords,omitempty"`
	Subtitle    *string    `xml:"itunes:subtitle,omitempty"`
	Summary     *string    `xml:"itunes:summary,omitempty"`

	parsed bool
}

var (
	ErrItem      = fmt.Errorf("%w: item", Err)
	ErrNoArticle = fmt.Errorf("%w: no article element found", ErrItem)
)

func NewItem(
	ctx context.Context,
	itemURL string,
) (*Item, error) {
	ap := &Item{}
	err := ap.parse(ctx, itemURL)
	if err != nil {
		return nil, err
	}

	return ap, nil
}

func (ap *Item) parse(ctx context.Context, itemURL string) error {
	if ap.parsed {
		return ErrAlreadyParsed
	}

	ap.URL = itemURL

	req, err := http.NewRequestWithContext(ctx, "GET", ap.URL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	ap.parseKeywords(doc)

	article := doc.Find("#article")
	if article.Length() != 1 {
		return ErrNoArticle
	}

	ap.parseTitle(article)
	ap.parseDescription(article)
	ap.parsePublishedDate(article)
	ap.parseAudioComponentContainer(ctx, article)

	ap.parsed = true

	return nil
}

func (item *Item) parseKeywords(s *goquery.Document) {
	tags, ok := s.Find("head meta[property='article:tag']").Attr("content")
	if !ok {
		return
	}

	kw := []string{}
	for _, tag := range strings.Split(tags, ",") {
		kw = append(kw, strings.TrimSpace(tag))
	}

	item.Keywords = strings.Join(kw, ", ")
}

func (item *Item) parseTitle(s *goquery.Selection) {
	elm := s.Find("h1[itemprop=headline]")
	if elm.Length() < 1 {
		return
	}

	innerText := elm.First().Text()
	innerText = strings.TrimSpace(innerText)
	innerText = strings.TrimSuffix(innerText, " – podcast")
	item.Title = innerText
}

func (item *Item) parseDescription(s *goquery.Selection) {
	content, ok := s.Find("meta[itemprop=description]").Attr("content")
	if ok {
		item.Description = strings.TrimSpace(content)
		item.Subtitle = &item.Description
		item.Summary = &item.Description
	}
}

func (item *Item) parsePublishedDate(s *goquery.Selection) {
	elm := s.Find("time[itemprop=datePublished]")
	timestamp, ok := elm.Attr("data-timestamp")
	if !ok {
		return
	}

	millisec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return
	}

	item.PublishDate = time.Unix(millisec/1000, millisec%1000)
}

func (item *Item) parseAudioComponentContainer(
	ctx context.Context,
	s *goquery.Selection,
) {
	figure := s.Find("#audio-component-container")
	if figure.Length() != 1 {
		return
	}

	guid, _ := figure.Attr("data-media-id")
	item.GUID = strings.TrimSpace(strings.TrimPrefix(guid, "gu-audio-"))

	duration, _ := figure.Attr("data-duration")
	if v, err := strconv.Atoi(duration); err == nil {
		d := time.Duration(v) * time.Second
		z := time.Unix(0, 0).UTC()
		item.Duration = z.Add(d).Format("15:04:05")
	}

	downloadURL, _ := figure.Attr("data-download-url")
	if strings.HasPrefix(downloadURL, "https://flex.acast.com/") {
		downloadURL = strings.Replace(
			downloadURL, "https://flex.acast.com/", "https://", 1,
		)
	}

	item.Enclosure, _ = NewEnclosure(ctx, downloadURL)
}
