// Copyright 2025 Ivan Guerreschi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const sampleRSS = `
<rss>
  <channel>
    <title>Test Feed</title>
    <description>Sample Description</description>
    <link>https://example.com/</link>
    <item>
      <title>Item 1</title>
      <link>https://example.com/1</link>
      <description><![CDATA[<p>Hello <b>world</b></p>]]></description>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>
`

func TestRemoveTags(t *testing.T) {
	reader, err := NewRssReader()
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	html := "<p>Hello <b>world</b></p>&nbsp;Test"
	expected := "Hello world Test"

	result := reader.removeTags(html)
	if result != expected {
		t.Errorf("removeTags() = %q, want %q", result, expected)
	}
}

func TestCategoryURLs(t *testing.T) {
	reader, err := NewRssReader()
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if _, ok := reader.categoryURLs[1]; !ok {
		t.Error("missing category 1")
	}
	if _, ok := reader.categoryURLs[3]; !ok {
		t.Error("missing category 3")
	}
	if _, ok := reader.categoryURLs[10]; ok {
		t.Error("unexpected category 10")
	}
}

func TestXMLParsing(t *testing.T) {
	var rss Rss
	err := xml.Unmarshal([]byte(sampleRSS), &rss)
	if err != nil {
		t.Fatalf("XML unmarshal failed: %v", err)
	}

	if rss.Channel.Title != "Test Feed" {
		t.Errorf("unexpected title: %s", rss.Channel.Title)
	}

	if len(rss.Channel.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(rss.Channel.Items))
	}
}

func TestFetchRssFeed(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, sampleRSS)
	}))
	defer s.Close()

	reader, _ := NewRssReader()
	ctx := context.Background()

	rss, err := reader.fetchRssFeed(ctx, s.URL)
	if err != nil {
		t.Fatalf("fetchRssFeed failed: %v", err)
	}

	if rss.Channel.Title != "Test Feed" {
		t.Errorf("unexpected feed title: %s", rss.Channel.Title)
	}
}

func TestFetchRssFeedStatusError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "err", http.StatusInternalServerError)
	}))
	defer s.Close()

	reader, _ := NewRssReader()
	ctx := context.Background()

	_, err := reader.fetchRssFeed(ctx, s.URL)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestFetchRssFeedMalformedXML(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<rss><channel><title>")
	}))
	defer s.Close()

	reader, _ := NewRssReader()
	ctx := context.Background()

	_, err := reader.fetchRssFeed(ctx, s.URL)
	if err == nil {
		t.Fatal("expected XML parse error")
	}
}

func TestGetCategoryInput(t *testing.T) {
	reader, _ := NewRssReader()

	tmp, err := os.CreateTemp("", "stdin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	input := "5\n"
	if _, err := tmp.WriteString(input); err != nil {
		t.Fatalf("failed to write to temp stdin file: %v", err)
	}

	if _, err := tmp.Seek(0, 0); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	oldStdin := os.Stdin
	os.Stdin = tmp
	defer func() { os.Stdin = oldStdin }()

	cat, err := reader.getCategoryInput()
	if err != nil {
		t.Fatalf("getCategoryInput error: %v", err)
	}

	if cat != 5 {
		t.Errorf("expected 5, got %d", cat)
	}
}
