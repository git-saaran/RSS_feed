package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// RSS feed structures
type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Category    string `xml:"category"`
	Source      string // We'll add this manually
}

// Advanced analytics structures
type NewsAnalytics struct {
	TotalArticles    int                    `json:"total_articles"`
	SourceCount      map[string]int         `json:"source_count"`
	CategoryCount    map[string]int         `json:"category_count"`
	HourlyCount      map[string]int         `json:"hourly_count"`
	SentimentScore   float64                `json:"sentiment_score"`
	TopKeywords      []KeywordCount         `json:"top_keywords"`
	TrendingTopics   []string               `json:"trending_topics"`
	Nifty50Mentions  int                    `json:"nifty50_mentions"`
	SourceReliability map[string]float64    `json:"source_reliability"`
}

type KeywordCount struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

type SentimentData struct {
	Positive float64 `json:"positive"`
	Neutral  float64 `json:"neutral"`
	Negative float64 `json:"negative"`
	Overall  string  `json:"overall"`
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for demo
	},
}

// WebSocket clients
var clients = make(map[*websocket.Conn]bool)
var clientsMutex sync.RWMutex

// NIFTY50 stocks list
var nifty50Stocks = []string{
	"RELIANCE", "TCS", "HDFCBANK", "INFY", "HINDUNILVR", "ICICIBANK", "ITC",
	"KOTAKBANK", "HCLTECH", "SBIN", "BHARTIARTL", "LTIM", "BAJFINANCE", "ADANIENT",
	"ASIANPAINT", "HINDALCO", "TITAN", "NTPC", "POWERGRID", "ULTRACEMCO", "M&M",
	"SUNPHARMA", "TATAMOTORS", "NESTLEIND", "BAJAJ-AUTO", "ADANIPORTS", "ADANIPOWER",
	"TATASTEEL", "JSWSTEEL", "BAJAJFINSV", "TECHM", "WIPRO", "HDFCLIFE", "GRASIM",
	"DIVISLAB", "APOLLOHOSP", "EICHERMOT", "BRITANNIA", "COALINDIA", "UPL", "TATACONSUM",
	"CIPLA", "SBILIFE", "MARUTI", "HDFC", "AXISBANK", "ONGC", "INDUSINDBK", "DRREDDY",
}

type NewsItem struct {
	Title           string        `json:"title"`
	Link            string        `json:"link"`
	Description     string        `json:"description"`
	PubDate         time.Time     `json:"pub_date"`
	TimeAgo         string        `json:"time_ago"`
	Category        string        `json:"category"`
	Source          string        `json:"source"`
	SourceColor     string        `json:"source_color"`
	SourceName      string        `json:"source_name"`
	HasNifty50      bool          `json:"has_nifty50"`
	Nifty50Stock    string        `json:"nifty50_stock"`
	SentimentScore  float64       `json:"sentiment_score"`
	SentimentLabel  string        `json:"sentiment_label"`
	Summary         string        `json:"summary"`
	Keywords        []string      `json:"keywords"`
	Priority        int           `json:"priority"`
	ReadingTime     int           `json:"reading_time"`
}

type NewsData struct {
	Items        []NewsItem     `json:"items"`
	LastUpdated  string         `json:"last_updated"`
	TotalSources int            `json:"total_sources"`
	Analytics    NewsAnalytics  `json:"analytics"`
	Sentiment    SentimentData  `json:"sentiment"`
}

// RSS feed sources
var rssSources = map[string]struct {
	URL   string
	Color string
	Name  string
}{
	"TOI": {
		URL:   "https://timesofindia.indiatimes.com/rssfeeds/1898055.cms",
		Color: "#dc2626",
		Name:  "Times of India Business",
	},
	"TH": {
		URL:   "https://www.thehindu.com/business/markets/feeder/default.rss",
		Color: "#dc2626",
		Name:  "The Hindu Business",
	},
	"BL": {
		URL:   "https://www.thehindubusinessline.com/markets/stock-markets/feeder/default.rss",
		Color: "#16a34a",
		Name:  "Business Line",
	},
	"LM": {
		URL:   "https://www.livemint.com/rss/markets",
		Color: "#0891b2",
		Name:  "LiveMint Markets",
	},
	"ZP": {
		URL:   "https://pulse.zerodha.com/feed.php",
		Color: "#7c3aed",
		Name:  "Zerodha Pulse",
	},
	"BS_MARKETS": {
		URL:   "https://www.business-standard.com/rss/markets-106.rss",
		Color: "#1e40af",
		Name:  "Business Standard - Markets",
	},
	"BS_NEWS": {
		URL:   "https://www.business-standard.com/rss/markets/news-10601.rss",
		Color: "#1e40af",
		Name:  "Business Standard - News",
	},
	"BS_COMMODITIES": {
		URL:   "https://www.business-standard.com/rss/markets/commodities-10608.rss",
		Color: "#1e40af",
		Name:  "Business Standard - Commodities",
	},
	"BS_IPO": {
		URL:   "https://www.business-standard.com/rss/markets/ipo-10611.rss",
		Color: "#1e40af",
		Name:  "Business Standard - IPO",
	},
	"BS_STOCK_MARKET": {
		URL:   "https://www.business-standard.com/rss/markets/stock-market-news-10618.rss",
		Color: "#1e40af",
		Name:  "Business Standard - Stock Market",
	},
	"BS_CRYPTO": {
		URL:   "https://www.business-standard.com/rss/markets/cryptocurrency-10622.rss",
		Color: "#1e40af",
		Name:  "Business Standard - Cryptocurrency",
	},
	"NSE_IT": {
		URL:   "https://nsearchives.nseindia.com/content/RSS/Insider_Trading.xml",
		Color: "#ea580c",
		Name:  "NSE Insider Trading",
	},
	"NSE_BB": {
		URL:   "https://nsearchives.nseindia.com/content/RSS/Daily_Buyback.xml",
		Color: "#ea580c",
		Name:  "NSE Daily Buy Back",
	},
	"NSE_FR": {
		URL:   "https://nsearchives.nseindia.com/content/RSS/Financial_Results.xml",
		Color: "#ea580c",
		Name:  "NSE Financial Results",
	},
	"NDTV_PROFIT": {
		URL:   "https://feeds.feedburner.com/ndtvprofit-latest",
		Color: "#1e40af",
		Name:  "NDTV Profit",
	},
}

// Real-time data structures (no historical storage)
var (
	currentNews   []NewsItem    // Only current batch, cleared on each refresh
	lastFetchTime time.Time
	newsMutex     sync.RWMutex
	liveAnalytics NewsAnalytics // Real-time analytics only
	liveSentiment SentimentData // Real-time sentiment only
)

// Configuration for memory efficiency
const (
	MAX_ARTICLES_PER_SOURCE = 10  // Limit articles per source
	MAX_TOTAL_ARTICLES      = 150 // Total articles limit (15 sources × 10)
	MEMORY_CLEANUP_INTERVAL = 1   // Cleanup every 1 minute
)

// Advanced AI-powered features
func analyzeSentiment(text string) (float64, string) {
	// Simple sentiment analysis based on keywords
	positiveWords := []string{"growth", "profit", "gain", "rise", "bull", "up", "surge", "boost", "positive", "strong", "high", "increase", "soar", "rally"}
	negativeWords := []string{"loss", "fall", "bear", "down", "decline", "drop", "crash", "weak", "low", "decrease", "plunge", "recession", "crisis"}
	
	text = strings.ToLower(text)
	positiveCount := 0
	negativeCount := 0
	
	for _, word := range positiveWords {
		if strings.Contains(text, word) {
			positiveCount++
		}
	}
	
	for _, word := range negativeWords {
		if strings.Contains(text, word) {
			negativeCount++
		}
	}
	
	score := float64(positiveCount-negativeCount) / float64(len(strings.Fields(text)))
	
	var label string
	if score > 0.1 {
		label = "Positive"
	} else if score < -0.1 {
		label = "Negative"
	} else {
		label = "Neutral"
	}
	
	return score, label
}

func extractKeywords(text string) []string {
	// Simple keyword extraction
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true, "in": true, "on": true, "at": true, "to": true, "for": true, "of": true, "with": true, "by": true, "is": true, "are": true, "was": true, "were": true, "will": true, "would": true, "could": true, "should": true, "may": true, "might": true, "can": true, "this": true, "that": true, "these": true, "those": true, "has": true, "have": true, "had": true,
	}
	
	text = strings.ToLower(text)
	re := regexp.MustCompile(`[^a-z\s]+`)
	text = re.ReplaceAllString(text, "")
	
	words := strings.Fields(text)
	keywords := []string{}
	
	for _, word := range words {
		if len(word) > 3 && !commonWords[word] {
			keywords = append(keywords, word)
		}
	}
	
	// Return first 5 keywords
	if len(keywords) > 5 {
		keywords = keywords[:5]
	}
	
	return keywords
}

func generateSummary(title, description string) string {
	// Simple extractive summarization
	sentences := strings.Split(description, ".")
	if len(sentences) > 2 {
		return sentences[0] + "."
	}
	return description
}

func calculateReadingTime(text string) int {
	words := len(strings.Fields(text))
	// Average reading speed: 200 words per minute
	return int(math.Ceil(float64(words) / 200.0))
}

func calculatePriority(item NewsItem) int {
	priority := 0
	
	// Higher priority for NIFTY50 mentions
	if item.HasNifty50 {
		priority += 30
	}
	
	// Higher priority for positive sentiment
	if item.SentimentScore > 0.1 {
		priority += 20
	} else if item.SentimentScore < -0.1 {
		priority += 15 // Negative news is also important
	}
	
	// Higher priority for recent news
	hoursSincePublication := time.Since(item.PubDate).Hours()
	if hoursSincePublication < 1 {
		priority += 25
	} else if hoursSincePublication < 6 {
		priority += 15
	} else if hoursSincePublication < 24 {
		priority += 10
	}
	
	// Higher priority for certain sources
	if strings.Contains(item.Source, "BS_") || item.Source == "LM" {
		priority += 10
	}
	
	return priority
}

func generateAnalytics(items []NewsItem) NewsAnalytics {
	analytics := NewsAnalytics{
		TotalArticles:    len(items),
		SourceCount:      make(map[string]int),
		CategoryCount:    make(map[string]int),
		HourlyCount:      make(map[string]int),
		SourceReliability: make(map[string]float64),
	}
	
	keywordCounts := make(map[string]int)
	var totalSentiment float64
	var niftyMentions int
	
	for _, item := range items {
		// Source count
		analytics.SourceCount[item.SourceName]++
		
		// Category count
		category := item.Category
		if category == "" {
			category = "General"
		}
		analytics.CategoryCount[category]++
		
		// Hourly distribution
		hour := item.PubDate.Format("15")
		analytics.HourlyCount[hour]++
		
		// Keywords
		for _, keyword := range item.Keywords {
			keywordCounts[keyword]++
		}
		
		// Sentiment
		totalSentiment += item.SentimentScore
		
		// NIFTY50 mentions
		if item.HasNifty50 {
			niftyMentions++
		}
		
		// Source reliability (based on sentiment and keywords quality)
		reliability := 0.5 + (item.SentimentScore * 0.2) + (float64(len(item.Keywords)) * 0.1)
		if reliability > 1.0 {
			reliability = 1.0
		}
		if reliability < 0.0 {
			reliability = 0.0
		}
		analytics.SourceReliability[item.SourceName] = reliability
	}
	
	// Calculate average sentiment
	if len(items) > 0 {
		analytics.SentimentScore = totalSentiment / float64(len(items))
	}
	
	analytics.Nifty50Mentions = niftyMentions
	
	// Top keywords
	type kv struct {
		Key   string
		Value int
	}
	
	var sortedKeywords []kv
	for k, v := range keywordCounts {
		sortedKeywords = append(sortedKeywords, kv{k, v})
	}
	
	sort.Slice(sortedKeywords, func(i, j int) bool {
		return sortedKeywords[i].Value > sortedKeywords[j].Value
	})
	
	for i, kv := range sortedKeywords {
		if i >= 10 { // Top 10 keywords
			break
		}
		analytics.TopKeywords = append(analytics.TopKeywords, KeywordCount{
			Keyword: kv.Key,
			Count:   kv.Value,
		})
	}
	
	// Generate trending topics (simplified)
	for _, kw := range analytics.TopKeywords[:min(5, len(analytics.TopKeywords))] {
		analytics.TrendingTopics = append(analytics.TrendingTopics, kw.Keyword)
	}
	
	return analytics
}

func generateSentimentData(items []NewsItem) SentimentData {
	var positive, neutral, negative int
	
	for _, item := range items {
		switch item.SentimentLabel {
		case "Positive":
			positive++
		case "Negative":
			negative++
		default:
			neutral++
		}
	}
	
	total := float64(len(items))
	if total == 0 {
		total = 1
	}
	
	sentimentData := SentimentData{
		Positive: float64(positive) / total * 100,
		Neutral:  float64(neutral) / total * 100,
		Negative: float64(negative) / total * 100,
	}
	
	// Determine overall sentiment
	if sentimentData.Positive > sentimentData.Negative && sentimentData.Positive > sentimentData.Neutral {
		sentimentData.Overall = "Positive"
	} else if sentimentData.Negative > sentimentData.Positive && sentimentData.Negative > sentimentData.Neutral {
		sentimentData.Overall = "Negative"
	} else {
		sentimentData.Overall = "Neutral"
	}
	
	return sentimentData
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// WebSocket handlers
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()
	
	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()
	
	log.Printf("Client connected. Total clients: %d", len(clients))
	
	// Send initial real-time data
	newsMutex.RLock()
	data := NewsData{
		Items:        currentNews,
		LastUpdated:  lastFetchTime.In(istLocation).Format("Jan 2, 2006 at 3:04 PM"),
		TotalSources: len(rssSources),
		Analytics:    liveAnalytics,
		Sentiment:    liveSentiment,
	}
	newsMutex.RUnlock()
	
	conn.WriteJSON(data)
	
	// Keep connection alive and handle disconnection
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			clientsMutex.Lock()
			delete(clients, conn)
			clientsMutex.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(clients))
			break
		}
	}
}

func broadcastUpdate() {
	newsMutex.RLock()
	data := NewsData{
		Items:        currentNews,
		LastUpdated:  lastFetchTime.In(istLocation).Format("Jan 2, 2006 at 3:04 PM"),
		TotalSources: len(rssSources),
		Analytics:    liveAnalytics,
		Sentiment:    liveSentiment,
	}
	newsMutex.RUnlock()
	
	clientsMutex.RLock()
	for client := range clients {
		err := client.WriteJSON(data)
		if err != nil {
			client.Close()
			delete(clients, client)
		}
	}
	clientsMutex.RUnlock()
}

func fetchRSSFeed(url string) (*RSS, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rss RSS
	err = xml.Unmarshal(body, &rss)
	if err != nil {
		return nil, err
	}

	return &rss, nil
}

// Load IST location
var istLocation *time.Location

func init() {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Printf("Warning: Could not load IST location, using local time: %v", err)
		loc = time.Local
	}
	istLocation = loc
}

func parseTime(dateStr string) time.Time {
	// Common RSS date formats
	formats := []string{
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 GMT",
		"Mon, 2 Jan 2006 15:04:05 GMT",
		"2006-01-02 15:04:05",
		"02-Jan-2006 15:04:05",     // Format used by Business Standard
		"02-Jan-2006 15:04",       // Format used by Business Standard (without seconds)
		"02-Jan-2006 15:04:05 MST", // With timezone
	}

	dateStr = strings.TrimSpace(dateStr)
	var t time.Time
	var err error

	// Try parsing with timezone first
	t, err = time.ParseInLocation("02-Jan-2006 15:04:05 MST", dateStr, istLocation)
	if err == nil {
		return t.In(istLocation)
	}

	// Try parsing without timezone
	t, err = time.ParseInLocation("02-Jan-2006 15:04:05", dateStr, istLocation)
	if err == nil {
		return t.In(istLocation)
	}

	// Try parsing without seconds
	t, err = time.ParseInLocation("02-Jan-2006 15:04", dateStr, istLocation)
	if err == nil {
		return t.In(istLocation)
	}

	// Try other standard formats
	for _, format := range formats {
		t, err := time.ParseInLocation(format, dateStr, istLocation)
		if err == nil {
			return t.In(istLocation)
		}
	}

	// If all parsing fails, return current time in IST
	if dateStr != "" && dateStr != "0000-00-00 00:00:00" {
		log.Printf("Failed to parse date: %s", dateStr)
	}
	return time.Now().In(istLocation)
}

func timeAgo(t time.Time) string {
	// Convert to IST if not already
	t = t.In(istLocation)
	now := time.Now().In(istLocation)
	duration := now.Sub(t)

	if duration < time.Minute {
		return "Just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}

func cleanDescription(desc string) string {
	// Remove CDATA tags
	desc = strings.ReplaceAll(desc, "<![CDATA[", "")
	desc = strings.ReplaceAll(desc, "]]>", "")

	// Simple HTML tag removal
	for strings.Contains(desc, "<") && strings.Contains(desc, ">") {
		start := strings.Index(desc, "<")
		end := strings.Index(desc[start:], ">")
		if end == -1 {
			break
		}
		desc = desc[:start] + desc[start+end+1:]
	}

	// Clean up extra whitespace
	desc = strings.ReplaceAll(desc, "\n", " ")
	desc = strings.ReplaceAll(desc, "\t", " ")
	for strings.Contains(desc, "  ") {
		desc = strings.ReplaceAll(desc, "  ", " ")
	}

	// Limit length
	if len(desc) > 180 {
		desc = desc[:180] + "..."
	}

	return strings.TrimSpace(desc)
}

// checkForNifty50 checks if the text contains any NIFTY50 stock mentions
func checkForNifty50(text string) (bool, string) {
	upperText := strings.ToUpper(text)
	for _, stock := range nifty50Stocks {
		if strings.Contains(upperText, stock) {
			return true, stock
		}
	}
	return false, ""
}

func fetchAllNews() {
	log.Println("🔄 Fetching real-time news (memory optimized)...")
	
	// Clear previous data for real-time operation
	newsMutex.Lock()
	currentNews = nil // Clear all previous news
	newsMutex.Unlock()
	
	var allNews []NewsItem
	var wg sync.WaitGroup
	var mu sync.Mutex

	for sourceName, source := range rssSources {
		wg.Add(1)
		go func(sName string, src struct {
			URL   string
			Color string
			Name  string
		}) {
			defer wg.Done()

			rss, err := fetchRSSFeed(src.URL)
			if err != nil {
				log.Printf("❌ Error fetching %s (%s): %v", sName, src.Name, err)
				return
			}

			// Limit articles per source for memory efficiency
			itemsToProcess := len(rss.Channel.Items)
			if itemsToProcess > MAX_ARTICLES_PER_SOURCE {
				itemsToProcess = MAX_ARTICLES_PER_SOURCE
				log.Printf("⚡ Limited %s to %d items (memory optimization)", sName, MAX_ARTICLES_PER_SOURCE)
			}

			log.Printf("✅ Fetched %s: processing %d/%d items", sName, itemsToProcess, len(rss.Channel.Items))

			mu.Lock()
			for i := 0; i < itemsToProcess; i++ {
				item := rss.Channel.Items[i]
				
				if item.Title == "" {
					continue // Skip empty items
				}

				pubTime := parseTime(item.PubDate)
				
				// Skip articles older than 24 hours for real-time focus
				if time.Since(pubTime) > 24*time.Hour {
					continue
				}

				// Check for NIFTY50 mentions in title and description
				hasNifty50Title, niftyStock := checkForNifty50(item.Title)
				hasNifty50Desc, niftyStockDesc := checkForNifty50(item.Description)
				hasNifty50 := hasNifty50Title || hasNifty50Desc
				niftyStockName := niftyStock
				if niftyStock == "" && niftyStockDesc != "" {
					niftyStockName = niftyStockDesc
				}

				// Lightweight processing for memory efficiency
				fullText := item.Title + " " + item.Description
				sentimentScore, sentimentLabel := analyzeSentiment(fullText)
				keywords := extractKeywords(fullText)
				summary := generateSummary(item.Title, item.Description)
				readingTime := calculateReadingTime(fullText)

				newsItem := NewsItem{
					Title:          item.Title,
					Link:           item.Link,
					Description:    cleanDescription(item.Description),
					PubDate:        pubTime,
					TimeAgo:        timeAgo(pubTime),
					Category:       item.Category,
					Source:         sName,
					SourceColor:    src.Color,
					SourceName:     src.Name,
					HasNifty50:     hasNifty50,
					Nifty50Stock:   niftyStockName,
					SentimentScore: sentimentScore,
					SentimentLabel: sentimentLabel,
					Summary:        summary,
					Keywords:       keywords,
					ReadingTime:    readingTime,
				}

				// Calculate priority
				newsItem.Priority = calculatePriority(newsItem)

				allNews = append(allNews, newsItem)
				
				// Memory safety check
				if len(allNews) >= MAX_TOTAL_ARTICLES {
					log.Printf("⚠️  Reached max articles limit (%d), stopping collection", MAX_TOTAL_ARTICLES)
					break
				}
			}
			mu.Unlock()
		}(sourceName, source)
	}

	wg.Wait()

	// Limit total articles and sort by priority + recency
	if len(allNews) > MAX_TOTAL_ARTICLES {
		log.Printf("⚡ Trimming to %d articles for memory efficiency", MAX_TOTAL_ARTICLES)
		
		// Sort by priority first, then by publication date (newest first)
		sort.Slice(allNews, func(i, j int) bool {
			if allNews[i].Priority == allNews[j].Priority {
				return allNews[i].PubDate.After(allNews[j].PubDate)
			}
			return allNews[i].Priority > allNews[j].Priority
		})
		
		// Keep only top articles
		allNews = allNews[:MAX_TOTAL_ARTICLES]
	}

	// Generate real-time analytics (no historical data)
	analyticsData := generateAnalytics(allNews)
	sentimentData := generateSentimentData(allNews)

	// Update real-time data (replace completely)
	newsMutex.Lock()
	currentNews = allNews
	lastFetchTime = time.Now()
	liveAnalytics = analyticsData
	liveSentiment = sentimentData
	newsMutex.Unlock()

	log.Printf("📊 Real-time articles: %d (max: %d)", len(allNews), MAX_TOTAL_ARTICLES)
	if len(analyticsData.TopKeywords) > 0 {
		log.Printf("🎯 Top keyword: %s", analyticsData.TopKeywords[0].Keyword)
	}
	log.Printf("😊 Live sentiment: %s", sentimentData.Overall)

	// Force garbage collection for memory efficiency
	runtime.GC()

	// Broadcast real-time update to WebSocket clients
	broadcastUpdate()
}

func getCurrentNews() ([]NewsItem, string) {
	newsMutex.RLock()
	defer newsMutex.RUnlock()

	// Update time ago for all items (real-time)
	for i := range currentNews {
		currentNews[i].TimeAgo = timeAgo(currentNews[i].PubDate)
	}

	// Format the time in IST
	istTime := lastFetchTime.In(istLocation)
	return currentNews, istTime.Format("Jan 2, 2006 at 3:04 PM")
}

// Real-time API handlers (no historical data)
func analyticsHandler(w http.ResponseWriter, r *http.Request) {
	newsMutex.RLock()
	data := liveAnalytics
	newsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(data)
}

func sentimentHandler(w http.ResponseWriter, r *http.Request) {
	newsMutex.RLock()
	data := liveSentiment
	newsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(data)
}

func filterHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	source := query.Get("source")
	category := query.Get("category")
	sentiment := query.Get("sentiment")
	nifty50Only := query.Get("nifty50") == "true"
	
	newsMutex.RLock()
	allItems := currentNews
	newsMutex.RUnlock()
	
	var filtered []NewsItem
	for _, item := range allItems {
		// Support BS_ALL for all Business Standard sources
		if source == "BS_ALL" {
			if !(item.Source == "BS_MARKETS" || item.Source == "BS_NEWS" || item.Source == "BS_COMMODITIES" || item.Source == "BS_IPO" || item.Source == "BS_CRYPTO") {
				continue
			}
		} else if source != "" && item.Source != source {
			continue
		}
		if category != "" && item.Category != category {
			continue
		}
		if sentiment != "" && item.SentimentLabel != sentiment {
			continue
		}
		if nifty50Only && !item.HasNifty50 {
			continue
		}
		filtered = append(filtered, item)
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(filtered)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	news, lastUpdated := getCurrentNews()
	
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Business Standard Feed</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --primary-color: #4f46e5;
            --bg-color: #ffffff;
            --text-color: #374151;
            --border-color: #e5e7eb;
            --hover-color: #f9fafb;
            --card-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
        }
        [data-theme="dark"] {
            --bg-color: #1f2937;
            --text-color: #f3f4f6;
            --border-color: #374151;
            --hover-color: #374151;
        }
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: 'Inter', sans-serif;
            background-color: var(--bg-color);
            color: var(--text-color);
            line-height: 1.5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 2rem;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid var(--border-color);
        }
        .bs-source-filter {
            margin-bottom: 2rem;
        }
        .bs-source-filter select {
            padding: 0.5rem 1rem;
            font-size: 1rem;
            border: 1px solid var(--border-color);
            border-radius: 0.5rem;
            background-color: var(--bg-color);
            color: var(--text-color);
            cursor: pointer;
            min-width: 200px;
        }
        .bs-source-filter select:focus {
            outline: none;
            border-color: var(--primary-color);
        }
        .news-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 1.5rem;
        }
        .news-card {
            background-color: var(--bg-color);
            border: 1px solid var(--border-color);
            border-radius: 0.5rem;
            padding: 1.5rem;
            box-shadow: var(--card-shadow);
            transition: transform 0.2s ease;
        }
        .news-card:hover {
            transform: translateY(-2px);
        }
        .news-source {
            font-size: 0.875rem;
            font-weight: 600;
            margin-bottom: 0.75rem;
            color: var(--primary-color);
        }
        .news-title {
            font-size: 1rem;
            font-weight: 600;
            margin-bottom: 0.75rem;
            color: var(--text-color);
            text-decoration: none;
        }
        .news-title:hover {
            color: var(--primary-color);
        }
        .news-description {
            font-size: 0.875rem;
            color: var(--text-color);
            margin-bottom: 1rem;
            opacity: 0.9;
        }
        .news-meta {
            display: flex;
            justify-content: space-between;
            align-items: center;
            font-size: 0.75rem;
            color: var(--text-color);
            opacity: 0.7;
        }
        @media (max-width: 768px) {
            .container {
                padding: 1rem;
            }
            .news-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header class="header">
            <h1>Business Standard Feed</h1>
            <div class="bs-source-filter">
                <select id="bsSourceFilter">
                    <option value="BS_ALL">All</option>
                    <option value="BS_MARKETS">Markets</option>
                    <option value="BS_NEWS">News</option>
                    <option value="BS_COMMODITIES">Commodities</option>
                    <option value="BS_IPO">IPO</option>
                    <option value="BS_CRYPTO">Cryptocurrency</option>
                </select>
            </div>
        </header>
        <div class="news-grid" id="bsNewsGrid">
            {{range .Items}}
            {{if or (eq .Source "BS_MARKETS") (eq .Source "BS_NEWS") (eq .Source "BS_COMMODITIES") (eq .Source "BS_IPO") (eq .Source "BS_CRYPTO")}}
            <article class="news-card" data-source="{{.Source}}">
                <div class="news-source">{{.SourceName}}</div>
                <a href="{{.Link}}" target="_blank" class="news-title">{{.Title}}</a>
                <p class="news-description">{{.Description}}</p>
                <div class="news-meta">
                    <span>{{.TimeAgo}}</span>
                </div>
            </article>
            {{end}}
            {{end}}
        </div>
    </div>
    <script>
        const bsSourceFilter = document.getElementById('bsSourceFilter');
        bsSourceFilter.addEventListener('change', (e) => {
            const val = e.target.value;
            window.location.href = val === 'BS_ALL' ? '/filter?source=BS_ALL' : `/filter?source=${val}`;
        });
    </script>
</body>
</html>
`

	t := template.Must(template.New("bsfeed").Parse(tmpl))
	t.Execute(w, struct{ Items []NewsItem }{news})
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	news, lastUpdated := getCurrentNews()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fmt.Fprintf(w, `{
		"items": %d,
		"last_updated": "%s",
		"sources": %d,
		"max_articles": %d,
		"memory_optimized": true,
		"status": "success"
	}`, len(news), lastUpdated, len(rssSources), MAX_TOTAL_ARTICLES)
}

// Memory management function
func performMemoryCleanup() {
	log.Println("🧹 Performing memory cleanup...")
	
	newsMutex.Lock()
	// Clear any articles older than 24 hours
	var recentNews []NewsItem
	cutoff := time.Now().Add(-24 * time.Hour)
	
	for _, item := range currentNews {
		if item.PubDate.After(cutoff) {
			recentNews = append(recentNews, item)
		}
	}
	
	if len(recentNews) != len(currentNews) {
		log.Printf("🗑️  Cleaned %d old articles", len(currentNews)-len(recentNews))
		currentNews = recentNews
	}
	newsMutex.Unlock()
	
	// Force garbage collection
	runtime.GC()
	
	// Log memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("💾 Memory: Alloc=%dKB Sys=%dKB NumGC=%d", 
		m.Alloc/1024, m.Sys/1024, m.NumGC)
}

func startPeriodicRefresh() {
	// Initial fetch
	fetchAllNews()

	// Set up periodic refresh every 5 minutes
	refreshTicker := time.NewTicker(5 * time.Minute)
	go func() {
		for range refreshTicker.C {
			fetchAllNews()
		}
	}()
	
	// Set up memory cleanup every 1 minute
	cleanupTicker := time.NewTicker(MEMORY_CLEANUP_INTERVAL * time.Minute)
	go func() {
		for range cleanupTicker.C {
			performMemoryCleanup()
		}
	}()
}

func main() {
	// Start the periodic refresh in the background
	go startPeriodicRefresh()

	// HTTP handlers
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/status", apiHandler)
	http.HandleFunc("/api/analytics", analyticsHandler)
	http.HandleFunc("/api/sentiment", sentimentHandler)
	http.HandleFunc("/api/filter", filterHandler)
	http.HandleFunc("/ws", handleWebSocket)

	fmt.Println("🚀 Advanced RSS News Aggregator starting...")
	fmt.Println("📡 Fetching feeds from", len(rssSources), "sources:")
	for code, source := range rssSources {
		fmt.Printf("   • %s: %s\n", code, source.Name)
	}
	fmt.Println("🔄 Auto-refresh interval: 5 minutes")
	fmt.Println("🌐 Server running at http://localhost:8080")
	fmt.Println("📊 API endpoints:")
	fmt.Println("   • Status: http://localhost:8080/api/status")
	fmt.Println("   • Analytics: http://localhost:8080/api/analytics")
	fmt.Println("   • Sentiment: http://localhost:8080/api/sentiment")
	fmt.Println("   • Filter: http://localhost:8080/api/filter")
	fmt.Println("🔌 WebSocket: ws://localhost:8080/ws")
	fmt.Println("🎯 Advanced Features:")
	fmt.Println("   • AI-powered sentiment analysis")
	fmt.Println("   • Real-time analytics dashboard")
	fmt.Println("   • Smart keyword extraction")
	fmt.Println("   • Priority-based news ranking")
	fmt.Println("   • Live WebSocket updates")

	log.Fatal(http.ListenAndServe(":8080", nil))
}