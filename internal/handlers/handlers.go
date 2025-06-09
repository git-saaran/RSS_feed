package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"rss_feed/internal/feed"
	"rss_feed/models"
	"rss_feed/pkg/logger"
)

// Handlers contains all the HTTP handlers
type Handlers struct {
	feedManager *feed.FeedManager
	logger      *logger.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(fm *feed.FeedManager, log *logger.Logger) *Handlers {
	return &Handlers{
		feedManager: fm,
		logger:      log,
	}
}

// HomeHandler handles the root endpoint
func (h *Handlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>RSS Feed Dashboard</title>
		<style>
			body { font-family: Arial, sans-serif; margin: 20px; }
			h1 { color: #333; }
			.news-item { margin-bottom: 20px; padding: 10px; border: 1px solid #ddd; border-radius: 4px; }
			.news-item h3 { margin-top: 0; }
			.source { color: #666; font-size: 0.9em; }
			.date { color: #888; font-size: 0.8em; }
		</style>
	</head>
	<body>
		<h1>RSS Feed Dashboard</h1>
		<div id="news">
			<p>Loading news...</p>
		</div>

		<script>
			// Fetch news from API
			fetch('/api/news?limit=10')
				.then(response => response.json())
				.then(data => {
					const newsContainer = document.getElementById('news');
					newsContainer.innerHTML = '';

					if (data.news && data.news.length > 0) {
						data.news.forEach(item => {
							const date = new Date(item.published);
							const newsItem = document.createElement('div');
							newsItem.className = 'news-item';
							newsItem.innerHTML = '\
								<h3><a href="' + item.link + '" target="_blank">' + item.title + '</a></h3>\
								<p>' + item.description + '</p>\
								<div class="source">' + item.source_name + ' • <span class="date">' + date.toLocaleString() + '</span></div>\
							';
							newsContainer.appendChild(newsItem);
						});
					} else {
						newsContainer.innerHTML = '<p>No news available.</p>';
					}
				})
				.catch(error => {
					console.error('Error fetching news:', error);
					document.getElementById('news').innerHTML = '<p>Error loading news. Please try again later.</p>';
				});
		</script>
	</body>
	</html>
	`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// HealthHandler handles health check requests
func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "ok",
		"version": "1.0.0",
		"time":    time.Now().Format(time.RFC3339),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// NewsHandler handles requests for news items
func (h *Handlers) NewsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	limit := parseInt(query.Get("limit"), 10)
	offset := parseInt(query.Get("offset"), 0)
	source := query.Get("source")
	category := query.Get("category")

	filter := models.FilterOptions{
		Limit:    limit,
		Offset:   offset,
		Source:   source,
		Category: category,
	}

	news, total := h.feedManager.GetNews(filter)

	response := map[string]interface{}{
		"news":       news,
		"total":      total,
		"returned":   len(news),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// FeedsHandler handles requests for feed information
func (h *Handlers) FeedsHandler(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would return the list of configured feeds
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"feeds": []map[string]interface{}{
			{
				"id":          "bbc",
				"name":        "BBC News",
				"url":         "http://feeds.bbci.co.uk/news/rss.xml",
				"description": "Latest news from BBC",
				"enabled":     true,
				"category":    "general",
			},
			{
				"id":          "reuters",
				"name":        "Reuters",
				"url":         "http://feeds.reuters.com/reuters/topNews",
				"description": "Latest news from Reuters",
				"enabled":     true,
				"category":    "general",
			},
		},
	})
}

// RefreshHandler handles requests to refresh feeds
func (h *Handlers) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would trigger a refresh of all feeds
	go func() {
		// This would be the actual refresh logic
		time.Sleep(2 * time.Second) // Simulate refresh time
	}()

	h.writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":  "accepted",
		"message": "Feed refresh started",
	})
}

// StatsHandler handles requests for dashboard statistics
func (h *Handlers) StatsHandler(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would return actual statistics
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_feeds":      2,
		"active_feeds":     2,
		"total_news_items": 20,
		"last_update_time": time.Now().Format(time.RFC3339),
		"uptime":           "1h23m45s",
	})
}

// writeJSON is a helper function to write JSON responses
func (h *Handlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Error encoding JSON response: %v", err)
	}
}

// parseInt parses a string to an integer, returning the default value if parsing fails
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}
