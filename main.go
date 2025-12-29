// Copyright 2025 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// -- Strutture XML per il parsing RSS --

// Rss represents the root <rss> element.
type Rss struct {
	Channel Channel `xml:"channel"`
}

// Channel represents the <channel> section of an RSS feed.
type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	Items       []Item `xml:"item"`
}

// Item represents a single <item> entry.
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// -- Strutture interne dell'applicazione --

// FeedCategory holds the metadata for a selectable RSS category.
type FeedCategory struct {
	ID   int
	Name string
	URL  string
}

// RssReader logic controller.
type RssReader struct {
	categories   []FeedCategory
	htmlTagRegex *regexp.Regexp
	client       *http.Client
}

// NewRssReader initializes the reader with configuration and compiled regex.
func NewRssReader() (*RssReader, error) {
	// Using a slice allows us to maintain order and generate the menu dynamically.
	categories := []FeedCategory{
		{1, "Prima Pagina", "https://www.adnkronos.com/RSS_PrimaPagina.xml"},
		{2, "Ultim'ora", "https://www.adnkronos.com/RSS_Ultimora.xml"},
		{3, "Politica", "https://www.adnkronos.com/RSS_Politica.xml"},
		{4, "Esteri", "https://www.adnkronos.com/RSS_Esteri.xml"},
		{5, "Cronaca", "https://www.adnkronos.com/RSS_Cronaca.xml"},
		{6, "Economia", "https://www.adnkronos.com/RSS_Economia.xml"},
		{7, "Finanza", "https://www.adnkronos.com/RSS_Finanza.xml"},
		{8, "Sport", "https://www.adnkronos.com/RSS_Sport.xml"},
	}

	// Compile regex once.
	re, err := regexp.Compile(`<[^>]*>`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex: %w", err)
	}

	return &RssReader{
		categories:   categories,
		htmlTagRegex: re,
		client:       &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// cleanText removes HTML tags and unescapes HTML entities.
func (r *RssReader) cleanText(text string) string {
	// 1. Remove HTML tags
	clean := r.htmlTagRegex.ReplaceAllString(text, "")
	// 2. Unescape entities (e.g. &quot; -> ", &agrave; -> à)
	clean = html.UnescapeString(clean)
	return strings.TrimSpace(clean)
}

// fetchFeed downloads and parses the RSS.
func (r *RssReader) fetchFeed(ctx context.Context, url string) (*Rss, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	
	// User-Agent is polite to add, though often optional
	req.Header.Set("User-Agent", "Adnkronos-CLI-Reader/1.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	var rss Rss
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, fmt.Errorf("xml decode error: %w", err)
	}

	return &rss, nil
}

// printMenu dynamically prints options based on the categories slice.
func (r *RssReader) printMenu() {
	fmt.Println("\n--- Adnkronos RSS Reader ---")
	fmt.Println("0: Esci")
	for _, cat := range r.categories {
		fmt.Printf("%d: %s\n", cat.ID, cat.Name)
	}
	fmt.Print("\nSeleziona un numero: ")
}

// displayFeed renders the feed items to stdout.
func (r *RssReader) displayFeed(rss *Rss) {
	fmt.Printf("\n=== %s ===\n", strings.ToUpper(rss.Channel.Title))
	fmt.Printf("%s\n\n", rss.Channel.Description)

	if len(rss.Channel.Items) == 0 {
		fmt.Println("Nessuna notizia trovata in questo feed.")
		return
	}

	for i, item := range rss.Channel.Items {
		fmt.Printf("[%d] %s\n", i+1, strings.TrimSpace(item.Title))
		// Optional: parse and format date nicely here if needed
		if item.PubDate != "" {
			fmt.Printf("    Pubblicato: %s\n", item.PubDate)
		}
		desc := r.cleanText(item.Description)
		if desc != "" {
			fmt.Printf("    %s\n", desc)
		}
		fmt.Println(strings.Repeat("-", 60))
	}
}

// Run starts the interactive loop.
func (r *RssReader) Run() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		r.printMenu()

		if !scanner.Scan() {
			break // EOF or error
		}
		input := strings.TrimSpace(scanner.Text())

		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println(">> Errore: Inserisci un numero valido.")
			continue
		}

		if choice == 0 {
			fmt.Println("Arrivederci!")
			return
		}

		// Find the selected category
		var selectedURL string
		for _, cat := range r.categories {
			if cat.ID == choice {
				selectedURL = cat.URL
				break
			}
		}

		if selectedURL == "" {
			fmt.Println(">> Errore: Categoria non valida.")
			continue
		}

		fmt.Println("Caricamento notizie in corso...")
		
		// Create a context with timeout for the fetch operation
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		rss, err := r.fetchFeed(ctx, selectedURL)
		cancel() // Cancel context as soon as fetch is done

		if err != nil {
			fmt.Printf(">> Errore nel scaricare il feed: %v\n", err)
			continue
		}

		r.displayFeed(rss)
		
		fmt.Println("\nPremi Invio per tornare al menu...")
		scanner.Scan() // Wait for user to read before clearing/showing menu again
	}
}

func main() {
	reader, err := NewRssReader()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Errore inizializzazione: %v\n", err)
		os.Exit(1)
	}

	reader.Run()
}
