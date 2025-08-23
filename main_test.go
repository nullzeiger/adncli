// Copyright 2025 Ivan Guerreschi <ivan.guerreschi.dev@gmail.com>.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRemoveTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic HTML tags",
			input:    "<p>Hello World</p>",
			expected: "Hello World",
		},
		{
			name:     "nested HTML tags",
			input:    "<div><p>Hello <b>World</b></p></div>",
			expected: "Hello World",
		},
		{
			name:     "HTML with attributes",
			input:    `<a href="https://example.com">Link</a>`,
			expected: "Link",
		},
		{
			name:     "special characters",
			input:    "Hello&nbsp;World",
			expected: "Hello World",
		},
		{
			name:     "mixed content",
			input:    "<p>Hello&nbsp;World</p><br/>Test",
			expected: "Hello WorldTest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeTags(tt.input)
			if result != tt.expected {
				t.Errorf("removeTags(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFetchRSSFeed(t *testing.T) {
	// Mock RSS feed
	mockRSS := `<?xml version="1.0" encoding="UTF-8"?>
	<rss version="2.0">
	<channel>
	<title>Test Feed</title>
	<description>Test Description</description>
	<link>https://example.com</link>
	<item>
	<title>Test Item</title>
	<link>https://example.com/item</link>
	<description>Test Item Description</description>
	<pubDate>Mon, 02 Jan 2025 15:04:05 GMT</pubDate>
	</item>
	</channel>
	</rss>`

	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		expectError    bool
	}{
		{
			name:           "valid RSS feed",
			serverResponse: mockRSS,
			statusCode:     http.StatusOK,
			expectError:    false,
		},
		{
			name:           "invalid XML",
			serverResponse: "invalid XML",
			statusCode:     http.StatusOK,
			expectError:    true,
		},
		{
			name:           "server error",
			serverResponse: "",
			statusCode:     http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Test fetchRSSFeed
			rss, err := fetchRSSFeed(ctx, server.URL)

			// Check error
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// For valid responses, verify RSS content
			if !tt.expectError && rss != nil {
				if rss.Channel.Title != "Test Feed" {
					t.Errorf("expected title 'Test Feed', got %q", rss.Channel.Title)
				}
				if len(rss.Channel.Items) != 1 {
					t.Errorf("expected 1 item, got %d", len(rss.Channel.Items))
				}
				if rss.Channel.Items[0].Title != "Test Item" {
					t.Errorf("expected item title 'Test Item', got %q", rss.Channel.Items[0].Title)
				}
			}
		})
	}
}

func TestCategoryURLs(t *testing.T) {
	tests := []struct {
		name        string
		category    int
		expectedURL string
		shouldExist bool
	}{
		{
			name:        "valid category - cronaca",
			category:    5,
			expectedURL: "https://www.adnkronos.com/RSS_Cronaca.xml",
			shouldExist: true,
		},
		{
			name:        "valid category - sport",
			category:    9,
			expectedURL: "https://www.adnkronos.com/RSS_Sport.xml",
			shouldExist: true,
		},
		{
			name:        "invalid category",
			category:    99,
			expectedURL: "",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, exists := categoryURLs[tt.category]

			if exists != tt.shouldExist {
				t.Errorf("category %d: expected exists=%v, got %v", tt.category, tt.shouldExist, exists)
			}

			if tt.shouldExist && url != tt.expectedURL {
				t.Errorf("category %d: expected URL %q, got %q", tt.category, tt.expectedURL, url)
			}
		})
	}
}

func TestContextTimeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("<rss><channel></channel></rss>"))
	}))
	defer server.Close()

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test should fail due to timeout
	_, err := fetchRSSFeed(ctx, server.URL)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") &&
		!strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}
