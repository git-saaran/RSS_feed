package models

import "time"

// Config defines the application configuration
type Config struct {
	Port            string        `json:"port"`
	PollInterval    time.Duration `json:"pollInterval"`
	RequestTimeout  time.Duration `json:"requestTimeout"`
	ServerTimeout   time.Duration `json:"serverTimeout"`
	MaxNewsItems    int           `json:"maxNewsItems"`
	EnableSentiment bool          `json:"enableSentiment"`
	LogLevel        string        `json:"logLevel"`
	DatabasePath    string        `json:"databasePath"`
	CacheTimeout    time.Duration `json:"cacheTimeout"`
	MaxConcurrent   int           `json:"maxConcurrent"`
	RateLimitRPM    int           `json:"rateLimitRPM"`
}

// FeedSource represents an RSS feed source
type FeedSource struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Logo         string    `json:"logo"`
	Class        string    `json:"class"`
	Enabled      bool      `json:"enabled"`
	Priority     int       `json:"priority"`
	Category     string    `json:"category"`
	Language     string    `json:"language"`
	Country      string    `json:"country"`
	UpdateFreq   int       `json:"updateFreq"`
	Status       string    `json:"status"`
	Error        string    `json:"error"`
	LastSync     time.Time `json:"lastSync"`
	SuccessCount int       `json:"successCount"`
	ErrorCount   int       `json:"errorCount"`
	AvgLatency   float64   `json:"avgLatency"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Description  string    `json:"description"`
	LastFetched  time.Time `json:"lastFetched"`
	LastError    string    `json:"lastError"`
}

// NewsItem represents a news article
type NewsItem struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Link         string    `json:"link"`
	Description  string    `json:"description"`
	PubDate      time.Time `json:"pubDate"`
	Published    time.Time `json:"published"`
	Category     string    `json:"category"`
	Source       string    `json:"source"`
	SourceID     string    `json:"sourceId"`
	SourceName   string    `json:"sourceName"`
	ImageURL     string    `json:"imageUrl"`
	Content      string    `json:"content"`
	Author       string    `json:"author"`
	Language     string    `json:"language"`
	Country      string    `json:"country"`
	Sentiment    float64   `json:"sentiment"`
	Score        float64   `json:"score"`
	Tags         []string  `json:"tags"`
	IsRead       bool      `json:"isRead"`
	IsBookmarked bool      `json:"isBookmarked"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// DashboardStats contains aggregated statistics
type DashboardStats struct {
	TotalNews      int           `json:"totalNews"`
	StockNews      int           `json:"stockNews"`
	ActiveFeeds    int           `json:"activeFeeds"`
	ErroredFeeds   int           `json:"erroredFeeds"`
	DisabledFeeds  int           `json:"disabledFeeds"`
	AvgLatency     float64       `json:"avgLatency"`
	TopSentiment   string        `json:"topSentiment"`
	CacheHitRate   float64       `json:"cacheHitRate"`
	MemoryUsageMB  float64       `json:"memoryUsageMB"`
	TotalFeeds     int           `json:"totalFeeds"`
	TotalNewsItems int           `json:"totalNewsItems"`
	LastUpdateTime time.Time     `json:"lastUpdateTime"`
	Uptime         time.Duration `json:"uptime"`
	RequestsServed int           `json:"requestsServed"`
	Errors         int           `json:"errors"`
}

// DashboardData contains all dashboard information
type DashboardData struct {
	Sources    []FeedSource   `json:"sources"`
	News       []NewsItem     `json:"news"`
	LastUpdate time.Time      `json:"lastUpdate"`
	LastUpdated time.Time     `json:"lastUpdated"`
	Stats      DashboardStats `json:"stats"`
}

// RateLimiter implements rate limiting for RSS feeds
type RateLimiter struct {
	Limit    int
	Interval time.Duration
	Tokens   int
	LastTime time.Time
}

// Cache implements a simple in-memory cache
type Cache struct {
	Data    map[string]interface{}
	Expires map[string]time.Time
	Hits    int
	Misses  int
	Timeout time.Duration
}

// RSS represents the RSS feed structure
type RSS struct {
	Channel struct {
		Items []Item `xml:"item"`
	} `xml:"channel"`
}

// Item represents an RSS feed item
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Category    string `xml:"category"`
	GUID        string `xml:"guid"`
}

// FilterOptions defines news filtering options
type FilterOptions struct {
	Source     string    `json:"source"`
	Category   string    `json:"category"`
	Sentiment  string    `json:"sentiment"`
	StockOnly  bool      `json:"stockOnly"`
	DateFrom   time.Time `json:"dateFrom"`
	DateTo     time.Time `json:"dateTo"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	MinScore   float64   `json:"minScore"`
	Keywords   []string  `json:"keywords"`
	SortBy     string    `json:"sortBy"`
	SortOrder  string    `json:"sortOrder"`
	Offset     int       `json:"offset"`
	Limit      int       `json:"limit"`
}
