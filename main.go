// Copyright 2025 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type Rss struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type RssReader struct {
	categoryURLs map[int]string
	htmlTagRegex *regexp.Regexp
	client       *http.Client
}

func NewRssReader() (*RssReader, error) {
	categoryURLs := map[int]string{
		1: "https://www.adnkronos.com/RSS_PrimaPagina.xml",
		2: "https://www.adnkronos.com/RSS_Ultimora.xml",
		3: "https://www.adnkronos.com/RSS_Politica.xml",
		4: "https://www.adnkronos.com/RSS_Esteri.xml",
		5: "https://www.adnkronos.com/RSS_Cronaca.xml",
		6: "https://www.adnkronos.com/RSS_Economia.xml",
		7: "https://www.adnkronos.com/RSS_Finanza.xml",
		8: "https://www.adnkronos.com/RSS_Sport.xml",
	}

	htmlTagRegex, err := regexp.Compile(`<[^>]*>`)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	return &RssReader{
		categoryURLs: categoryURLs,
		htmlTagRegex: htmlTagRegex,
		client:       client,
	}, nil
}

func (r *RssReader) removeTags(text string) string {
	clean := r.htmlTagRegex.ReplaceAllString(text, "")
	clean = strings.ReplaceAll(clean, "&nbsp;", " ")
	return strings.TrimSpace(clean)
}

func (r *RssReader) fetchRssFeed(ctx context.Context, url string) (*Rss, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var rss Rss
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, err
	}

	return &rss, nil
}

func (r *RssReader) printMenu() {
	fmt.Println("Adnkronos RSS Reader")
	fmt.Println("0: Exit")
	fmt.Println("1: Prima Pagina")
	fmt.Println("2: Ultim'ora")
	fmt.Println("3: Politica")
	fmt.Println("4: Esteri")
	fmt.Println("5: Cronaca")
	fmt.Println("6: Economia")
	fmt.Println("7: Finanza")
	fmt.Println("8: Sport")
	fmt.Print("\nSelect category number: ")
}

func (r *RssReader) getCategoryInput() (int, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}

	input = strings.TrimSpace(input)
	var cat int
	_, err = fmt.Sscanf(input, "%d", &cat)
	return cat, err
}

func (r *RssReader) displayFeed(rss *Rss) {
	fmt.Printf("\nTitle: %s\n", rss.Channel.Title)
	fmt.Printf("Link: %s\n", rss.Channel.Link)
	fmt.Printf("Description: %s\n\n", rss.Channel.Description)

	for _, item := range rss.Channel.Items {
		fmt.Printf("Title: %s\n", item.Title)
		fmt.Printf("Link: %s\n", item.Link)
		fmt.Printf("Description: %s\n", r.removeTags(item.Description))
		fmt.Printf("Published: %s\n\n", item.PubDate)
		fmt.Println(strings.Repeat("-", 80))
	}
}

func (r *RssReader) Run(ctx context.Context) error {
	r.printMenu()

	category, err := r.getCategoryInput()
	if err != nil {
		return err
	}

	if category == 0 {
		os.Exit(0)
	}

	url, ok := r.categoryURLs[category]
	if !ok {
		return fmt.Errorf("invalid category number")
	}

	rss, err := r.fetchRssFeed(ctx, url)
	if err != nil {
		return err
	}

	r.displayFeed(rss)
	return nil
}

func main() {
	reader, err := NewRssReader()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if err := reader.Run(ctx); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)

	}
}
