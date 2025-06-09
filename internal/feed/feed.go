package feed

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"rss_feed/config"
	"rss_feed/models"
	"rss_feed/pkg/logger"
	"rss_feed/pkg/cache"
	"rss_feed/pkg/ratelimit"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html/charset"
)

// RSS represents the root element of an RSS feed
type RSS struct {
	Channel Channel `xml:"channel"`
}

// Channel represents the channel element in an RSS feed
type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

// Item represents an item in an RSS feed
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// FeedManager manages RSS feeds
type FeedManager struct {
	config      *config.Config
	logger      *logger.Logger
	feeds       map[string]*models.FeedSource
	news        []models.NewsItem
	lastUpdate  time.Time
	stats       models.DashboardStats
	mu          sync.RWMutex
	rateLimiter *ratelimit.RateLimiter
	cache       *cache.Cache
	client      *http.Client
}

// NewFeedManager creates a new FeedManager
func NewFeedManager(cfg *config.Config, log *logger.Logger) *FeedManager {
	return &FeedManager{
		config:      cfg,
		logger:      log,
		feeds:       GetDefaultFeedSources(),
		news:        make([]models.NewsItem, 0),
		lastUpdate:  time.Now(),
		rateLimiter: ratelimit.NewRateLimiter(cfg.RateLimitRPM),
		cache:       cache.NewCache(cfg.CacheTimeout),
		client: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
}

// GetDefaultFeedSources returns a map of default feed sources
func GetDefaultFeedSources() map[string]*models.FeedSource {
	return map[string]*models.FeedSource{
		"bbc": {
			ID:          "bbc",
			Name:        "BBC News",
			URL:         "http://feeds.bbci.co.uk/news/rss.xml",
			Description: "Latest news from BBC",
			Enabled:     true,
			Category:    "general",
		},
		"reuters": {
			ID:          "reuters",
			Name:        "Reuters",
			URL:         "http://feeds.reuters.com/reuters/topNews",
			Description: "Latest news from Reuters",
			Enabled:     true,
			Category:    "general",
		},
	}
}

// Start begins the feed update process
func (fm *FeedManager) Start(ctx context.Context) {
	fm.logger.Info("Starting feed manager")
	
	// Initial update
	fm.UpdateAllFeeds(ctx)


	// Start periodic updates
	ticker := time.NewTicker(fm.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fm.UpdateAllFeeds(ctx)
		case <-ctx.Done():
			fm.logger.Info("Stopping feed manager")
			return
		}
	}
}

// UpdateAllFeeds updates all enabled feeds
func (fm *FeedManager) UpdateAllFeeds(ctx context.Context) {
	fm.logger.Info("Updating all feeds")

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allNews []models.NewsItem

	for _, source := range fm.feeds {
		if !source.Enabled {
			continue
		}

		wg.Add(1)
		go func(src *models.FeedSource) {
			defer wg.Done()

			fm.rateLimiter.Wait()

			items, err := fm.fetchRSSFeed(ctx, src)
			if err != nil {
				fm.logger.Error("Error fetching feed %s: %v", src.Name, err)
				src.LastError = err.Error()
				return
			}

			mu.Lock()
			allNews = append(allNews, items...)
			src.LastFetched = time.Now()
			src.LastError = ""
			mu.Unlock()

		}(source)
	}

	wg.Wait()

	if len(allNews) > 0 {
		fm.mu.Lock()
		fm.news = append(allNews, fm.news...)
		if len(fm.news) > fm.config.MaxNewsItems {
			fm.news = fm.news[:fm.config.MaxNewsItems]
		}
		fm.lastUpdate = time.Now()
		fm.updateStats()
		fm.mu.Unlock()

		fm.logger.Info("Updated %d news items", len(allNews))
	}
}

// fetchRSSFeed fetches and parses an RSS feed
func (fm *FeedManager) fetchRSSFeed(ctx context.Context, source *models.FeedSource) ([]models.NewsItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	resp, err := fm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching feed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Handle character encoding
	reader, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("error creating charset reader: %v", err)
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("error parsing XML: %v", err)
	}

	var newsItems []models.NewsItem
	for _, item := range rss.Channel.Items {
		pubDate, _ := time.Parse(time.RFC1123Z, item.PubDate)
		if pubDate.IsZero() {
			pubDate = time.Now()
		}

		newsItem := models.NewsItem{
			ID:          item.GUID,
			Title:       strings.TrimSpace(item.Title),
			Description: strings.TrimSpace(item.Description),
			Link:        item.Link,
			Published:   pubDate,
			Source:      source.ID,
			SourceName:  source.Name,
			Category:    source.Category,
		}

		newsItems = append(newsItems, newsItem)
	}

	return newsItems, nil
}

// GetDashboardData returns data for the dashboard
func (fm *FeedManager) GetDashboardData() models.DashboardData {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	return models.DashboardData{
		News:        fm.getLatestNews(10), // Get 10 latest news items
		Stats:       fm.stats,
		LastUpdated: fm.lastUpdate,
	}
}

// GetNews returns news items based on filter options
func (fm *FeedManager) GetNews(filter models.FilterOptions) ([]models.NewsItem, int) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	var filtered []models.NewsItem

	for _, item := range fm.news {
		if filter.Source != "" && item.Source != filter.Source {
			continue
		}
		if filter.Category != "" && item.Category != filter.Category {
			continue
		}
		if !filter.StartTime.IsZero() && item.Published.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && item.Published.After(filter.EndTime) {
			continue
		}

		filtered = append(filtered, item)
	}

	// Sort by published date (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Published.After(filtered[j].Published)
	})

	total := len(filtered)

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(filtered) {
		filtered = filtered[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(filtered) {
		filtered = filtered[:filter.Limit]
	}

	return filtered, total
}

// getLatestNews returns the latest n news items
func (fm *FeedManager) getLatestNews(limit int) []models.NewsItem {
	if limit <= 0 || limit > len(fm.news) {
		limit = len(fm.news)
	}

	return fm.news[:limit]
}

// updateStats updates the dashboard statistics
func (fm *FeedManager) updateStats() {
	activeFeeds := 0
	for _, feed := range fm.feeds {
		if feed.Enabled {
			activeFeeds++
		}
	}

	fm.stats = models.DashboardStats{
		TotalFeeds:      len(fm.feeds),
		ActiveFeeds:     activeFeeds,
		TotalNewsItems:  len(fm.news),
		LastUpdateTime:  time.Now(),
		Uptime:          time.Since(fm.stats.LastUpdateTime) + fm.stats.Uptime,
		RequestsServed:  fm.stats.RequestsServed,
		Errors:          fm.stats.Errors,
	}
}
