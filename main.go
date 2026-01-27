// Copyright 2026 Ivan Guerreschi. All rights reserved.
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

// --- ANSI Color Codes ---
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

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
	clean := r.htmlTagRegex.ReplaceAllString(text, "")
	clean = html.UnescapeString(clean)
	return strings.TrimSpace(clean)
}

// fetchFeed downloads and parses the RSS.
func (r *RssReader) fetchFeed(ctx context.Context, url string) (*Rss, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

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
	// Header in Bold Cyan
	fmt.Printf("\n%s--- Adnkronos RSS Reader ---%s\n", ColorBold+ColorCyan, ColorReset)

	// Option 0 in Red
	fmt.Printf("%s0:%s Esci\n", ColorRed, ColorReset)

	for _, cat := range r.categories {
		// ID in Yellow, Name in standard color
		fmt.Printf("%s%d:%s %s\n", ColorYellow, cat.ID, ColorReset, cat.Name)
	}
	// Prompt in Bold
	fmt.Printf("\n%sSeleziona un numero: %s", ColorBold, ColorReset)
}

// displayFeed renders the feed items to stdout.
func (r *RssReader) displayFeed(rss *Rss) {
	// Channel Title in Bold Green background or just Bold Green text
	fmt.Printf("\n%s=== %s ===%s\n", ColorBold+ColorGreen, strings.ToUpper(rss.Channel.Title), ColorReset)
	fmt.Printf("%s%s%s\n\n", ColorPurple, rss.Channel.Description, ColorReset)

	if len(rss.Channel.Items) == 0 {
		fmt.Println("Nessuna notizia trovata in questo feed.")
		return
	}

	for i, item := range rss.Channel.Items {
		// Index in Blue, Title in Bold White
		fmt.Printf("%s[%d]%s %s%s%s\n", ColorBlue, i+1, ColorReset, ColorBold, strings.TrimSpace(item.Title), ColorReset)

		if item.PubDate != "" {
			// Date in Cyan
			fmt.Printf("    Pubblicato: %s%s%s\n", ColorCyan, item.PubDate, ColorReset)
		}

		desc := r.cleanText(item.Description)
		if desc != "" {
			fmt.Printf("    %s\n", desc)
		}

		// Separator in faint gray (using standard here for compatibility)
		fmt.Println(strings.Repeat("-", 60))
	}
}

// Run starts the interactive loop.
func (r *RssReader) Run() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		r.printMenu()

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		choice, err := strconv.Atoi(input)
		if err != nil {
			// Error in Red
			fmt.Printf("%s>> Errore: Inserisci un numero valido.%s\n", ColorRed, ColorReset)
			continue
		}

		if choice == 0 {
			fmt.Println("Arrivederci!")
			return
		}

		var selectedURL string
		for _, cat := range r.categories {
			if cat.ID == choice {
				selectedURL = cat.URL
				break
			}
		}

		if selectedURL == "" {
			fmt.Printf("%s>> Errore: Categoria non valida.%s\n", ColorRed, ColorReset)
			continue
		}

		fmt.Println("Caricamento notizie in corso...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		rss, err := r.fetchFeed(ctx, selectedURL)
		cancel()

		if err != nil {
			fmt.Printf("%s>> Errore nel scaricare il feed: %v%s\n", ColorRed, err, ColorReset)
			continue
		}

		r.displayFeed(rss)
	}
}

func main() {
	reader, err := NewRssReader()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sErrore inizializzazione: %v%s\n", ColorRed, err, ColorReset)
		os.Exit(1)
	}

	reader.Run()
}
