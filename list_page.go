package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ListPage struct {
	URL         *url.URL
	PrevPageURL *url.URL
	NextPageURL *url.URL
	ArticleURLs []string

	parsed bool
}

func NewListPage(
	ctx context.Context,
	u *url.URL,
) (*ListPage, error) {
	ip := &ListPage{URL: u}

	err := ip.Parse(ctx)
	if err != nil {
		return nil, err
	}

	return ip, nil
}

func (lp *ListPage) Parse(ctx context.Context) error {
	if lp.parsed {
		return ErrAlreadyParsed
	}

	req, err := http.NewRequestWithContext(ctx, "GET", lp.URL.String(), nil)
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

	lp.ArticleURLs = lp.extractArticleURLs(doc)

	lp.NextPageURL, err = lp.extractNextPageURL(doc)
	if err != nil {
		return err
	}

	lp.PrevPageURL, err = lp.extractPrevPageURL(doc)
	if err != nil {
		return err
	}

	lp.parsed = true

	return nil
}

func (lp *ListPage) extractArticleURLs(doc *goquery.Document) []string {
	uniqURLs := map[string]bool{}

	doc.Find("section a[data-link-name=article]").Each(
		func(_ int, s *goquery.Selection) {
			href, ok := s.Attr("href")
			if ok && href != "" {
				uniqURLs[strings.TrimSpace(href)] = true
			}
		},
	)

	urls := []string{}
	for u := range uniqURLs {
		urls = append(urls, u)
	}

	return urls
}

func (lp *ListPage) extractNextPageURL(doc *goquery.Document) (*url.URL, error) {
	href, ok := doc.Find(".pagination a[rel=next]").Attr("href")
	if !ok {
		return nil, nil
	}

	u, err := url.Parse(href)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (lp *ListPage) extractPrevPageURL(doc *goquery.Document) (*url.URL, error) {
	href, ok := doc.Find(".pagination a[rel=prev]").Attr("href")
	if !ok {
		return nil, nil
	}

	u, err := url.Parse(href)
	if err != nil {
		return nil, err
	}

	return u, nil
}
