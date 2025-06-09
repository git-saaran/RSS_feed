package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html/charset"
)

type FeedManager struct {
	config      *Config
	logger      *Logger
	feeds       map[string]FeedSource
	news        []NewsItem
	lastUpdate  time.Time
	stats       DashboardStats
	mutex       sync.RWMutex
	rateLimiter *RateLimiter
	cache       *Cache
	client      *http.Client
}

func NewFeedManager(config *Config, logger *Logger) *FeedManager {
	return &FeedManager{
		config:      config,
		logger:      logger,
		feeds:       GetDefaultFeedSources(),
		news:        make([]NewsItem, 0),
		lastUpdate:  time.Now(),
		stats:       DashboardStats{},
		rateLimiter: NewRateLimiter(config.RateLimitRPM),
		cache:       NewCache(config.CacheTimeout),
		client: &http.Client{
			Timeout: config.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:          20,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       30 * time.Second,
				DisableCompression:    false,
				DisableKeepAlives:     false,
				MaxConnsPerHost:       10,
				ResponseHeaderTimeout: 10 * time.Second,
			},
		},
	}
}

func (fm *FeedManager) Start(ctx context.Context) {
	fm.logger.Info("Starting Feed Manager")

	// Initial fetch
	fm.UpdateAllFeeds(ctx)

	// Start periodic updates
	ticker := time.NewTicker(fm.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fm.UpdateAllFeeds(ctx)
		case <-ctx.Done():
			fm.logger.Info("Feed Manager stopped")
			return
		}
	}
}

func (fm *FeedManager) UpdateAllFeeds(ctx context.Context) {
	startTime := time.Now()
	fm.logger.Info("Starting feed update cycle")

	var wg sync.WaitGroup
	var allNews []NewsItem
	var newsMutex sync.Mutex
	semaphore := make(chan struct{}, fm.config.MaxConcurrent)

	fm.mutex.RLock()
	feeds := make(map[string]FeedSource)
	for k, v := range fm.feeds {
		if v.Enabled {
			feeds[k] = v
		}
	}
	fm.mutex.RUnlock()

	for key, source := range feeds {
		wg.Add(1)
		go func(key string, source FeedSource) {
			defer wg.Done()

			// Rate limiting
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check rate limiter
			if !fm.rateLimiter.Allow() {
				fm.logger.Warn("Rate limit exceeded for %s", source.Name)
				return
			}

			// Update source status
			fm.updateSourceStatus(key, "loading", "")

			// Create context with timeout
			feedCtx, cancel := context.WithTimeout(ctx, fm.config.RequestTimeout)
			defer cancel()

			// Try cache first
			cacheKey := fmt.Sprintf("feed_%s", key)
			if cachedNews, found := fm.cache.Get(cacheKey); found {
				if newsItems, ok := cachedNews.([]NewsItem); ok {
					fm.logger.Debug("Using cached data for %s", source.Name)
					newsMutex.Lock()
					allNews = append(allNews, newsItems...)
					newsMutex.Unlock()
					fm.updateSourceStatus(key, "success", "")
					return
				}
			}

			// Fetch fresh data
			newsItems, err := fm.fetchRSSFeed(feedCtx, source)
			if err != nil {
				fm.logger.Error("Error fetching %s: %v", source.Name, err)
				fm.updateSourceStatus(key, "error", err.Error())
				return
			}

			// Cache the results
			fm.cache.Set(cacheKey, newsItems)

			// Add to collection
			newsMutex.Lock()
			allNews = append(allNews, newsItems...)
			newsMutex.Unlock()

			fm.updateSourceStatus(key, "success", "")
			fm.logger.Debug("Successfully fetched %d items from %s", len(newsItems), source.Name)

		}(key, source)
	}

	wg.Wait()

	// Process and update news
	fm.processNews(allNews)

	duration := time.Since(startTime)
	fm.logger.Info("Feed update completed: %d items from %d sources in %v",
		len(allNews), len(feeds), duration)
}

func (fm *FeedManager) fetchRSSFeed(ctx context.Context, source FeedSource) ([]NewsItem, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Enhanced headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FinancialNewsBot/2.0)")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := fm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Read body with size limit (10MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if response is HTML instead of RSS
	bodyStr := strings.ToLower(string(body[:min(1000, len(body))]))
	if strings.Contains(bodyStr, "<!doctype html") ||
		strings.Contains(bodyStr, "<html") ||
		strings.Contains(bodyStr, "<title>") {
		return nil, fmt.Errorf("received HTML response instead of RSS feed")
	}

	// Parse RSS
	decoder := xml.NewDecoder(strings.NewReader(string(body)))
	decoder.CharsetReader = charset.NewReaderLabel

	var rss RSS
	if err := decoder.Decode(&rss); err != nil {
		return nil, fmt.Errorf("failed to parse RSS: %w", err)
	}

	// Convert RSS items to NewsItems
	var newsItems []NewsItem
	for _, item := range rss.Channel.Items {
		if strings.TrimSpace(item.Title) == "" {
			continue
		}

		newsItem := fm.createNewsItem(item, source)
		newsItems = append(newsItems, newsItem)
	}

	return newsItems, nil
}

func (fm *FeedManager) createNewsItem(item Item, source FeedSource) NewsItem {
	pubDate := ParseDate(item.PubDate)
	stockSymbols := ExtractStockSymbols(item.Title + " " + item.Description)
	cleanTitle := CleanText(item.Title)
	cleanDesc := CleanText(item.Description)

	newsItem := NewsItem{
		ID:           GenerateID(item.GUID, item.Link, item.Title),
		Title:        cleanTitle,
		Link:         item.Link,
		Description:  cleanDesc,
		PubDate:      pubDate,
		Category:     item.Category,
		Source:       source.Name,
		IsStockNews:  len(stockSymbols) > 0,
		TimeAgo:      GetTimeAgo(pubDate),
		StockSymbols: stockSymbols,
		WordCount:    CountWords(cleanTitle + " " + cleanDesc),
		ReadTime:     CalculateReadTime(cleanTitle + " " + cleanDesc),
		Score:        CalculateScore(cleanTitle, cleanDesc, stockSymbols, source.Priority),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Add sentiment analysis if enabled
	if fm.config.EnableSentiment {
		newsItem.Sentiment = AnalyzeSentiment(item.Title + " " + item.Description)
	}

	// Extract tags
	newsItem.Tags = ExtractTags(newsItem.Title, newsItem.Description)

	return newsItem
}

func (fm *FeedManager) processNews(allNews []NewsItem) {
	// Remove duplicates
	allNews = RemoveDuplicates(allNews)

	// Sort by publication date (newest first)
	sort.Slice(allNews, func(i, j int) bool {
		return allNews[i].PubDate.After(allNews[j].PubDate)
	})

	// Limit news items
	if len(allNews) > fm.config.MaxNewsItems {
		allNews = allNews[:fm.config.MaxNewsItems]
	}

	// Update time ago for all items
	for i := range allNews {
		allNews[i].TimeAgo = GetTimeAgo(allNews[i].PubDate)
	}

	// Update data with thread safety
	fm.mutex.Lock()
	fm.news = allNews
	fm.lastUpdate = time.Now()
	fm.updateStats()
	fm.mutex.Unlock()
}

func (fm *FeedManager) updateSourceStatus(key, status, errorMsg string) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	if source, exists := fm.feeds[key]; exists {
		source.Status = status
		source.Error = errorMsg
		source.UpdatedAt = time.Now()

		if status == "success" {
			source.LastSync = time.Now()
			source.SuccessCount++
		} else if status == "error" {
			source.ErrorCount++
		}

		fm.feeds[key] = source
	}
}

func (fm *FeedManager) updateStats() {
	activeFeeds := 0
	erroredFeeds := 0
	disabledFeeds := 0
	stockNews := 0
	totalLatency := 0.0
	sentimentCounts := make(map[string]int)

	for _, source := range fm.feeds {
		if !source.Enabled {
			disabledFeeds++
			continue
		}

		switch source.Status {
		case "success":
			activeFeeds++
		case "error":
			erroredFeeds++
		}

		totalLatency += source.AvgLatency
	}

	for _, news := range fm.news {
		if news.IsStockNews {
			stockNews++
		}
		if news.Sentiment != "" {
			sentimentCounts[news.Sentiment]++
		}
	}

	// Find top sentiment
	topSentiment := "neutral"
	maxCount := sentimentCounts["neutral"]
	for sentiment, count := range sentimentCounts {
		if count > maxCount {
			maxCount = count
			topSentiment = sentiment
		}
	}

	fm.stats = DashboardStats{
		TotalNews:     len(fm.news),
		StockNews:     stockNews,
		ActiveFeeds:   activeFeeds,
		ErroredFeeds:  erroredFeeds,
		DisabledFeeds: disabledFeeds,
		AvgLatency:    totalLatency / float64(len(fm.feeds)),
		TopSentiment:  topSentiment,
		CacheHitRate:  fm.cache.GetHitRate(),
		MemoryUsageMB: GetMemoryUsage(),
	}
}

func (fm *FeedManager) GetDashboardData() DashboardData {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	// Convert feeds map to slice
	var sources []FeedSource
	for _, source := range fm.feeds {
		sources = append(sources, source)
	}

	return DashboardData{
		Sources:    sources,
		News:       fm.news,
		LastUpdate: fm.lastUpdate,
		Stats:      fm.stats,
	}
}

func (fm *FeedManager) GetNews(filter FilterOptions) ([]NewsItem, int) {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	return FilterNews(fm.news, filter)
}

func (fm *FeedManager) RefreshFeed(feedID string) error {
	fm.mutex.RLock()
	source, exists := fm.feeds[feedID]
	fm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("feed not found: %s", feedID)
	}

	if !source.Enabled {
		return fmt.Errorf("feed is disabled: %s", feedID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), fm.config.RequestTimeout)
	defer cancel()

	// Clear cache for this feed
	cacheKey := fmt.Sprintf("feed_%s", feedID)
	fm.cache.Delete(cacheKey)

	newsItems, err := fm.fetchRSSFeed(ctx, source)
	if err != nil {
		fm.updateSourceStatus(feedID, "error", err.Error())
		return err
	}

	fm.updateSourceStatus(feedID, "success", "")
	fm.logger.Info("Manually refreshed feed %s: %d items", source.Name, len(newsItems))

	return nil
}

func (fm *FeedManager) EnableFeed(feedID string) error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	if source, exists := fm.feeds[feedID]; exists {
		source.Enabled = true
		source.UpdatedAt = time.Now()
		fm.feeds[feedID] = source
		return nil
	}
	return fmt.Errorf("feed not found: %s", feedID)
}

func (fm *FeedManager) DisableFeed(feedID string) error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	if source, exists := fm.feeds[feedID]; exists {
		source.Enabled = false
		source.UpdatedAt = time.Now()
		fm.feeds[feedID] = source
		return nil
	}
	return fmt.Errorf("feed not found: %s", feedID)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
