// Copyright 2025 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package main provides a simple command-line RSS reader for
// Adnkronos RSS feeds. It allows users to select a news category,
// fetch the corresponding RSS feed, and display parsed content.
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

// Rss represents the root <rss> element of an RSS document.
type Rss struct {
	Channel Channel `xml:"channel"`
}

// Channel represents the <channel> section of an RSS feed,
// containing metadata and a collection of RSS items.
type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	Items       []Item `xml:"item"`
}

// Item represents a single <item> entry in an RSS feed, containing
// information about an individual news article.
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// RssReader provides the functionality to load RSS feeds,
// remove HTML tags from descriptions, and display the parsed results.
type RssReader struct {
	categoryURLs map[int]string // Mapping of category numbers to RSS URLs
	htmlTagRegex *regexp.Regexp // Precompiled regex for HTML tag removal
	client       *http.Client   // HTTP client used to fetch RSS feeds
}

// NewRssReader initializes and returns a configured RssReader.
// It prepares the RSS category URL map, compiles required regular
// expressions, and sets up an HTTP client.
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

	client := &http.Client{Timeout: 15 * time.Second}

	return &RssReader{
		categoryURLs: categoryURLs,
		htmlTagRegex: htmlTagRegex,
		client:       client,
	}, nil
}

// removeTags strips all HTML tags from the given text and also replaces
// common HTML entities such as &nbsp;. This is used to clean RSS descriptions.
func (r *RssReader) removeTags(text string) string {
	clean := r.htmlTagRegex.ReplaceAllString(text, "")
	clean = strings.ReplaceAll(clean, "&nbsp;", " ")
	return strings.TrimSpace(clean)
}

// fetchRssFeed downloads and parses an RSS feed from the specified URL.
// It performs the HTTP request using the provided context, validates the
// response, and decodes the XML body into an Rss struct.
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

// printMenu displays the available RSS categories to the user.
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

// getCategoryInput reads user input from stdin and parses it into
// an integer representing the selected RSS category.
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

// displayFeed prints the high-level channel information followed by
// each individual RSS item, including cleaned descriptions.
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

// Run starts the RSS reader: it displays the category menu,
// retrieves the user's selection, fetches the chosen RSS feed,
// and outputs its parsed content.
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

// main initializes the RssReader and executes it. If any error occurs,
// the program prints an error message and terminates with a non-zero code.
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
