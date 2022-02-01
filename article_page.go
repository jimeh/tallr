package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type ArticlePage struct {
	URL *url.URL

	Title       string
	Description string
	Enclosure   string
	PublishDate time.Time
	GUID        string
	Duration    time.Duration

	parsed bool
}

var (
	ErrArticlePage = fmt.Errorf("%w: article page", Err)
	ErrNoArticle   = fmt.Errorf("%w: no article element found", ErrArticlePage)
)

func NewArticlePage(
	ctx context.Context,
	u *url.URL,
) (*ArticlePage, error) {
	ap := &ArticlePage{URL: u}

	err := ap.Parse(ctx)
	if err != nil {
		return nil, err
	}

	return ap, nil
}

func (ap *ArticlePage) Parse(ctx context.Context) error {
	if ap.parsed {
		return ErrAlreadyParsed
	}

	req, err := http.NewRequestWithContext(ctx, "GET", ap.URL.String(), nil)
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

	article := doc.Find("#article")
	if article.Length() != 1 {
		return ErrNoArticle
	}

	ap.Title = ap.extractTitle(article)
	ap.Description = ap.extractDescription(article)

	if v, err := ap.extractPublishedDate(article); err == nil {
		ap.PublishDate = v
	}

	figure := article.Find("#audio-component-container")
	if figure.Length() == 1 {
		guid, _ := figure.Attr("data-media-id")
		ap.GUID = strings.TrimSpace(strings.TrimPrefix(guid, "gu-audio-"))

		ap.Enclosure, _ = figure.Attr("data-download-url")

		duration, _ := figure.Attr("data-duration")
		if v, err := strconv.Atoi(duration); err == nil {
			ap.Duration = time.Duration(v) * time.Second
		}
	}

	ap.parsed = true

	return nil
}

func (ap *ArticlePage) extractTitle(s *goquery.Selection) string {
	elm := s.Find("h1[itemprop=headline]")
	if elm.Length() < 1 {
		return ""
	}

	innerText := elm.Text()
	innerText = strings.TrimSpace(innerText)
	innerText = strings.TrimSuffix(innerText, " – podcast")

	return innerText
}

func (ap *ArticlePage) extractDescription(s *goquery.Selection) string {
	content, ok := s.Find("meta[itemprop=description]").Attr("content")
	if !ok {
		return ""
	}

	return strings.TrimSpace(content)
}

func (ap *ArticlePage) extractPublishedDate(s *goquery.Selection) (time.Time, error) {
	elm := s.Find("time[itemprop=datePublished]")
	timestamp, ok := elm.Attr("data-timestamp")
	if !ok {
		return time.Time{}, nil
	}
	fmt.Printf("timestamp: %+v\n", timestamp)
	millisec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	pubDate := time.Unix(millisec/1000, millisec%1000)

	return pubDate, nil
}
