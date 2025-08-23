// Copyright 2025 Ivan Guerreschi <ivan.guerreschi.dev@gmail.com>.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

// RSS represents the root XML structure of an RSS feed
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

// Channel represents the main content container in an RSS feed
type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	Items       []Item `xml:"item"`
}

// Item represents a single article in the RSS feed
type Item struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	Desc    string `xml:"description"`
	PubDate string `xml:"pubDate"`
}

var (
	// categoryURLs maps category numbers to their respective RSS feed URLs
	categoryURLs = map[int]string{
		1: "https://www.adnkronos.com/RSS_PrimaPagina.xml",
		2: "https://www.adnkronos.com/RSS_Ultimora.xml",
		3: "https://www.adnkronos.com/RSS_Politica.xml",
		4: "https://www.adnkronos.com/RSS_Esteri.xml",
		5: "https://www.adnkronos.com/RSS_Cronaca.xml",
		6: "https://www.adnkronos.com/RSS_Economia.xml",
		7: "https://www.adnkronos.com/RSS_Finanza.xml",
		8: "https://www.adnkronos.com/RSS_Tecnologia.xml",
		9: "https://www.adnkronos.com/RSS_Sport.xml",
	}

	// htmlTagRegex matches HTML tags for removal
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
)

// removeTags removes HTML tags and special characters from the input text
func removeTags(text string) string {
	// Remove HTML tags using regex
	cleanText := htmlTagRegex.ReplaceAllString(text, "")

	// Remove special characters
	cleanText = regexp.MustCompile(`&nbsp;`).ReplaceAllString(cleanText, " ")

	return cleanText
}

// fetchRSSFeed retrieves and parses the RSS feed from the given URL
func fetchRSSFeed(ctx context.Context, url string) (*RSS, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching RSS feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var rss RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, fmt.Errorf("decoding XML: %w", err)
	}

	return &rss, nil
}

// printMenu displays the available RSS categories
func printMenu() {
	fmt.Println("AGI RSS Reader")
	fmt.Println("0: Exit")
	fmt.Println("1: Prima Pagina")
	fmt.Println("2: Ultim'ora")
	fmt.Println("3: Politica")
	fmt.Println("4: Esteri")
	fmt.Println("5: Cronaca")
	fmt.Println("6: Economia")
	fmt.Println("7: Finanza")
	fmt.Println("8: Tecnologia")
	fmt.Println("9: Sport")
	fmt.Print("\nSelect category number: ")
}

func main() {
	printMenu()

	var category int
	if _, err := fmt.Scan(&category); err != nil {
		log.Fatal("Error reading input:", err)
	}

	if category == 0 {
		os.Exit(0)
	}

	url, exists := categoryURLs[category]
	if !exists {
		log.Fatal("Invalid category number")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rss, err := fetchRSSFeed(ctx, url)
	if err != nil {
		log.Fatal("Error fetching RSS feed:", err)
	}

	// Print feed information
	fmt.Printf("\nTitle: %s\n", rss.Channel.Title)
	fmt.Printf("Description: %s\n\n", rss.Channel.Description)

	// Print each item
	for _, item := range rss.Channel.Items {
		fmt.Printf("Title: %s\n", item.Title)
		fmt.Printf("Link: %s\n", item.Link)
		fmt.Printf("Description: %s\n", removeTags(item.Desc))
		fmt.Printf("Published: %s\n\n", item.PubDate)
		fmt.Println("--------------------------------------------------------------------------------")
	}
}
