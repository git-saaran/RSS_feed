package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
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

// News aggregator structure
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
	Title        string
	Link         string
	Description  string
	PubDate      time.Time
	TimeAgo      string
	Category     string
	Source       string
	SourceColor  string
	SourceName   string
	HasNifty50   bool   // Flag for NIFTY50 stock mention
	Nifty50Stock string // The actual NIFTY50 stock mentioned (if any)
}

type NewsData struct {
	Items        []NewsItem
	GroupedItems map[string][]NewsItem
	Sources      map[string]struct {
		URL   string
		Color string
		Name  string
	}
	LastUpdated  string
	TotalSources int
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

// Global cache for news items
var (
	newsCache     []NewsItem
	lastCacheTime time.Time
	cacheMutex    sync.RWMutex
)

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
	log.Println("🔄 Fetching news from all sources...")
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

			log.Printf("✅ Successfully fetched %s: %d items", sName, len(rss.Channel.Items))

			mu.Lock()
			for _, item := range rss.Channel.Items {
				if item.Title == "" {
					continue // Skip empty items
				}

				pubTime := parseTime(item.PubDate)

				// Check for NIFTY50 mentions in title and description
				hasNifty50Title, niftyStock := checkForNifty50(item.Title)
				hasNifty50Desc, niftyStockDesc := checkForNifty50(item.Description)
				hasNifty50 := hasNifty50Title || hasNifty50Desc
				niftyStockName := niftyStock
				if niftyStock == "" && niftyStockDesc != "" {
					niftyStockName = niftyStockDesc
				}

				newsItem := NewsItem{
					Title:        item.Title,
					Link:         item.Link,
					Description:   cleanDescription(item.Description),
					PubDate:      pubTime,
					TimeAgo:      timeAgo(pubTime),
					Category:     item.Category,
					Source:       sName,
					SourceColor:  src.Color,
					SourceName:   src.Name,
					HasNifty50:   hasNifty50,
					Nifty50Stock: niftyStockName,
				}

				allNews = append(allNews, newsItem)
			}
			mu.Unlock()
		}(sourceName, source)
	}

	wg.Wait()

	// Sort by publication date (newest first)
	sort.Slice(allNews, func(i, j int) bool {
		return allNews[i].PubDate.After(allNews[j].PubDate)
	})

	// Update cache
	cacheMutex.Lock()
	newsCache = allNews
	lastCacheTime = time.Now()
	cacheMutex.Unlock()

	log.Printf("📊 Total news items cached: %d", len(allNews))
}

func getNewsFromCache() ([]NewsItem, string) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	// Update time ago for all items
	for i := range newsCache {
		newsCache[i].TimeAgo = timeAgo(newsCache[i].PubDate)
	}

	// Format the time in IST
	istTime := lastCacheTime.In(istLocation)
	return newsCache, istTime.Format("Jan 2, 2006 at 3:04 PM")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	news, lastUpdated := getNewsFromCache()

	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>📈 Business News Aggregator</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --primary-gradient: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            --dark-gradient: linear-gradient(135deg, #1e293b 0%, #334155 100%);
            --card-bg: #ffffff;
            --card-bg-dark: #1e293b;
            --text-primary: #1f2937;
            --text-primary-dark: #f8fafc;
            --text-secondary: #6b7280;
            --text-secondary-dark: #94a3b8;
            --border-color: #e5e7eb;
            --border-color-dark: #374151;
            --highlight-color: #3b82f6;
            --success-color: #10b981;
            --warning-color: #fbbf24;
            --nifty-bg: #fff9e6;
            --nifty-bg-dark: #451a03;
            --nifty-border: #ffd700;
            --search-bg: rgba(255, 255, 255, 0.1);
            --search-bg-dark: rgba(0, 0, 0, 0.2);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            background: var(--primary-gradient);
            min-height: 100vh;
            padding: 20px;
            transition: all 0.3s ease;
            color: var(--text-primary);
        }

        body.dark {
            background: var(--dark-gradient);
            color: var(--text-primary-dark);
        }
        
        .container {
            max-width: 1800px;
            margin: 0 auto;
        }
        
        .header {
            text-align: center;
            margin-bottom: 40px;
            animation: fadeInUp 0.8s ease;
        }
        
        .header h1 {
            color: white;
            font-size: clamp(2rem, 5vw, 3rem);
            margin-bottom: 15px;
            text-shadow: 0 4px 8px rgba(0,0,0,0.3);
            font-weight: 700;
            letter-spacing: -0.02em;
        }
        
        .header p {
            color: rgba(255,255,255,0.9);
            font-size: 1.2rem;
            margin-bottom: 10px;
            font-weight: 400;
        }
        
        .last-updated {
            color: rgba(255,255,255,0.8);
            font-size: 0.95rem;
            font-style: italic;
            font-weight: 300;
        }

        .controls {
            display: flex;
            justify-content: center;
            align-items: center;
            gap: 20px;
            margin-bottom: 30px;
            flex-wrap: wrap;
            animation: fadeInUp 0.8s ease 0.2s both;
        }

        .search-container {
            position: relative;
            min-width: 300px;
        }

        .search-input {
            width: 100%;
            padding: 12px 45px 12px 45px;
            border: none;
            border-radius: 25px;
            background: var(--search-bg);
            backdrop-filter: blur(10px);
            color: white;
            font-size: 16px;
            transition: all 0.3s ease;
            outline: none;
        }

        .search-input::placeholder {
            color: rgba(255, 255, 255, 0.7);
        }

        .search-input:focus {
            background: rgba(255, 255, 255, 0.2);
            transform: scale(1.02);
        }

        .search-icon, .clear-icon {
            position: absolute;
            top: 50%;
            transform: translateY(-50%);
            color: rgba(255, 255, 255, 0.8);
            font-size: 18px;
        }

        .search-icon {
            left: 15px;
        }

        .clear-icon {
            right: 15px;
            cursor: pointer;
            display: none;
            transition: color 0.2s;
        }

        .clear-icon:hover {
            color: white;
        }

        .filter-container {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
        }

        .filter-btn, .theme-toggle {
            padding: 10px 20px;
            border: 2px solid rgba(255, 255, 255, 0.3);
            border-radius: 25px;
            background: transparent;
            color: white;
            cursor: pointer;
            transition: all 0.3s ease;
            font-size: 14px;
            font-weight: 500;
            white-space: nowrap;
        }

        .filter-btn:hover, .theme-toggle:hover {
            background: rgba(255, 255, 255, 0.2);
            transform: translateY(-2px);
        }

        .filter-btn.active {
            background: white;
            color: var(--highlight-color);
            border-color: white;
        }

        .theme-toggle {
            font-size: 18px;
            padding: 10px 15px;
        }
        
        .stats-bar {
            display: flex;
            justify-content: center;
            gap: 20px;
            margin-bottom: 30px;
            flex-wrap: wrap;
            animation: fadeInUp 0.8s ease 0.4s both;
        }
        
        .stat-item {
            background: rgba(255,255,255,0.15);
            color: white;
            padding: 12px 20px;
            border-radius: 20px;
            font-size: 14px;
            backdrop-filter: blur(15px);
            border: 1px solid rgba(255,255,255,0.2);
            transition: all 0.3s ease;
            font-weight: 500;
        }

        .stat-item:hover {
            background: rgba(255,255,255,0.25);
            transform: translateY(-2px);
        }

        .loading-container {
            display: none;
            text-align: center;
            margin: 50px 0;
            animation: fadeIn 0.5s ease;
        }

        .loading-spinner {
            width: 50px;
            height: 50px;
            border: 4px solid rgba(255, 255, 255, 0.3);
            border-top: 4px solid white;
            border-radius: 50%;
            margin: 20px auto;
            animation: spin 1s linear infinite;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        .news-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 25px;
            animation: fadeInUp 0.8s ease 0.6s both;
        }
        
        .news-source {
            background: var(--card-bg);
            border-radius: 20px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.1);
            overflow: hidden;
            backdrop-filter: blur(10px);
            transition: all 0.3s ease;
            border: 1px solid var(--border-color);
        }

        body.dark .news-source {
            background: var(--card-bg-dark);
            border-color: var(--border-color-dark);
            box-shadow: 0 10px 40px rgba(0,0,0,0.3);
        }

        .news-source:hover {
            transform: translateY(-5px);
            box-shadow: 0 20px 60px rgba(0,0,0,0.15);
        }

        body.dark .news-source:hover {
            box-shadow: 0 20px 60px rgba(0,0,0,0.4);
        }
        
        .source-header {
            padding: 20px 24px;
            display: flex;
            align-items: center;
            gap: 15px;
            border-bottom: 1px solid var(--border-color);
            background: linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%);
        }

        body.dark .source-header {
            border-bottom-color: var(--border-color-dark);
            background: linear-gradient(135deg, rgba(255,255,255,0.05) 0%, rgba(255,255,255,0.02) 100%);
        }
        
        .source-icon {
            width: 45px;
            height: 45px;
            border-radius: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: 700;
            font-size: 12px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.2);
        }
        
        .source-name {
            font-weight: 600;
            color: var(--text-primary);
            flex: 1;
            font-size: 15px;
        }

        body.dark .source-name {
            color: var(--text-primary-dark);
        }
        
        .updated-badge {
            background: var(--success-color);
            color: white;
            padding: 6px 12px;
            border-radius: 15px;
            font-size: 12px;
            font-weight: 600;
            box-shadow: 0 2px 8px rgba(16, 185, 129, 0.3);
        }
        
        .item-count {
            background: var(--text-secondary);
            color: white;
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 11px;
            margin-left: 8px;
            font-weight: 600;
        }
        
        .news-items {
            max-height: 600px;
            overflow-y: auto;
        }
        
        .news-item {
            padding: 16px 24px;
            border-bottom: 1px solid var(--border-color);
            transition: all 0.3s ease;
            position: relative;
            border-left: 3px solid transparent;
        }

        body.dark .news-item {
            border-bottom-color: var(--border-color-dark);
        }
        
        .nifty50-highlight {
            background-color: var(--nifty-bg);
            border-left-color: var(--nifty-border);
            box-shadow: 0 2px 12px rgba(255, 215, 0, 0.15);
        }

        body.dark .nifty50-highlight {
            background-color: var(--nifty-bg-dark);
        }
        
        .nifty50-badge {
            position: absolute;
            top: 12px;
            right: 20px;
            background: var(--nifty-border);
            color: #000;
            padding: 4px 10px;
            border-radius: 12px;
            font-size: 10px;
            font-weight: 700;
            text-transform: uppercase;
            box-shadow: 0 2px 8px rgba(255, 215, 0, 0.3);
            z-index: 1;
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0%, 100% { transform: scale(1); }
            50% { transform: scale(1.05); }
        }
        
        .news-item:hover {
            background-color: rgba(59, 130, 246, 0.05);
            transform: translateX(5px);
        }

        body.dark .news-item:hover {
            background-color: rgba(59, 130, 246, 0.1);
        }
        
        .news-item:last-child {
            border-bottom: none;
        }
        
        .news-title {
            font-weight: 600;
            color: var(--text-primary);
            margin-bottom: 8px;
            line-height: 1.4;
            text-decoration: none;
            display: block;
            font-size: 15px;
            transition: color 0.2s ease;
        }

        body.dark .news-title {
            color: var(--text-primary-dark);
        }
        
        .news-title:hover {
            color: var(--highlight-color);
        }
        
        .news-description {
            color: var(--text-secondary);
            font-size: 13px;
            line-height: 1.5;
            margin-bottom: 12px;
        }

        body.dark .news-description {
            color: var(--text-secondary-dark);
        }
        
        .news-meta {
            display: flex;
            justify-content: space-between;
            align-items: center;
            font-size: 12px;
            color: var(--text-secondary);
        }

        body.dark .news-meta {
            color: var(--text-secondary-dark);
        }
        
        .news-time {
            font-weight: 500;
            display: flex;
            align-items: center;
            gap: 5px;
        }
        
        .news-category {
            background: rgba(59, 130, 246, 0.1);
            color: var(--highlight-color);
            padding: 4px 10px;
            border-radius: 10px;
            font-weight: 500;
            font-size: 11px;
        }

        body.dark .news-category {
            background: rgba(59, 130, 246, 0.2);
        }
        
        .action-buttons {
            position: fixed;
            bottom: 30px;
            right: 30px;
            display: flex;
            flex-direction: column;
            gap: 15px;
            z-index: 1000;
        }

        .action-btn {
            background: var(--highlight-color);
            color: white;
            border: none;
            border-radius: 50%;
            width: 60px;
            height: 60px;
            cursor: pointer;
            box-shadow: 0 6px 20px rgba(59, 130, 246, 0.4);
            transition: all 0.3s ease;
            font-size: 20px;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        .action-btn:hover {
            background: #2563eb;
            transform: scale(1.1);
        }

        .action-btn:active {
            transform: scale(0.95);
        }

        .scroll-top {
            background: var(--success-color);
            box-shadow: 0 6px 20px rgba(16, 185, 129, 0.4);
        }

        .scroll-top:hover {
            background: #059669;
        }
        
        .loading {
            text-align: center;
            color: white;
            font-size: 1.2rem;
            margin: 50px 0;
        }

        .no-results {
            text-align: center;
            color: white;
            font-size: 1.1rem;
            margin: 50px 0;
            opacity: 0.8;
        }

        .filter-count {
            display: inline-block;
            background: rgba(255, 255, 255, 0.2);
            color: white;
            padding: 4px 8px;
            border-radius: 10px;
            font-size: 12px;
            margin-left: 8px;
            font-weight: 500;
        }
        
        @media (max-width: 1024px) {
            .news-grid {
                grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
                gap: 20px;
            }
        }

        @media (max-width: 768px) {
            body {
                padding: 15px;
            }

            .news-grid {
                grid-template-columns: 1fr;
                gap: 15px;
            }
            
            .header h1 {
                font-size: 2.2rem;
            }

            .header p {
                font-size: 1rem;
            }
            
            .stats-bar, .controls {
                gap: 10px;
            }
            
            .stat-item {
                font-size: 12px;
                padding: 8px 14px;
            }

            .search-container {
                min-width: 250px;
            }

            .filter-container {
                justify-content: center;
            }

            .action-buttons {
                bottom: 20px;
                right: 20px;
            }

            .action-btn {
                width: 50px;
                height: 50px;
                font-size: 18px;
            }

            .news-source {
                border-radius: 15px;
            }

            .source-header {
                padding: 16px 20px;
            }

            .news-item {
                padding: 14px 20px;
            }
        }

        @media (max-width: 480px) {
            .controls {
                flex-direction: column;
                align-items: stretch;
            }

            .search-container {
                min-width: auto;
            }

            .filter-container {
                justify-content: center;
            }

            .filter-btn, .theme-toggle {
                flex: 1;
                text-align: center;
            }
        }
        
        /* Enhanced Scrollbar styling */
        .news-items::-webkit-scrollbar {
            width: 8px;
        }
        
        .news-items::-webkit-scrollbar-track {
            background: rgba(0,0,0,0.05);
            border-radius: 4px;
        }
        
        .news-items::-webkit-scrollbar-thumb {
            background: rgba(0,0,0,0.2);
            border-radius: 4px;
            transition: background 0.2s;
        }
        
        .news-items::-webkit-scrollbar-thumb:hover {
            background: rgba(0,0,0,0.3);
        }

        body.dark .news-items::-webkit-scrollbar-track {
            background: rgba(255,255,255,0.05);
        }

        body.dark .news-items::-webkit-scrollbar-thumb {
            background: rgba(255,255,255,0.2);
        }

        body.dark .news-items::-webkit-scrollbar-thumb:hover {
            background: rgba(255,255,255,0.3);
        }

        /* Animations */
        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }

        @keyframes fadeInUp {
            from {
                opacity: 0;
                transform: translateY(30px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        @keyframes slideIn {
            from {
                opacity: 0;
                transform: translateX(-20px);
            }
            to {
                opacity: 1;
                transform: translateX(0);
            }
        }

        .news-item {
            animation: slideIn 0.5s ease forwards;
        }

        .news-item:nth-child(even) {
            animation-delay: 0.1s;
        }

        /* Accessibility improvements */
        .action-btn:focus,
        .filter-btn:focus,
        .theme-toggle:focus,
        .search-input:focus {
            outline: 2px solid rgba(255, 255, 255, 0.8);
            outline-offset: 2px;
        }

        /* Reduced motion for accessibility */
        @media (prefers-reduced-motion: reduce) {
            * {
                animation-duration: 0.01ms !important;
                animation-iteration-count: 1 !important;
                transition-duration: 0.01ms !important;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📈 Business News Aggregator</h1>
            <p>Real-time updates from {{.TotalSources}} premium financial sources</p>
            <div class="last-updated">Last updated: {{.LastUpdated}}</div>
        </div>

        <div class="controls">
            <div class="search-container">
                <span class="search-icon">🔍</span>
                <input type="text" class="search-input" id="searchInput" placeholder="Search news articles...">
                <span class="clear-icon" id="clearSearch">✕</span>
            </div>
            <div class="filter-container">
                <button class="filter-btn active" data-filter="all">All <span class="filter-count" id="allCount">{{len .Items}}</span></button>
                <button class="filter-btn" data-filter="nifty50">NIFTY50 <span class="filter-count" id="niftyCount">0</span></button>
                <button class="filter-btn" data-filter="recent">Recent <span class="filter-count" id="recentCount">0</span></button>
                <button class="theme-toggle" id="themeToggle" title="Toggle Dark Mode">🌙</button>
            </div>
        </div>
        
        <div class="stats-bar">
            <div class="stat-item">📊 Total Articles: <span id="totalCount">{{len .Items}}</span></div>
            <div class="stat-item">🔄 Auto-refresh: 5 min</div>
            <div class="stat-item">📡 Live Sources: {{.TotalSources}}</div>
            <div class="stat-item">⭐ NIFTY50 Mentions: <span id="niftyMentions">0</span></div>
        </div>

        <div class="loading-container" id="loadingContainer">
            <div class="loading-spinner"></div>
            <div>Loading news...</div>
        </div>

        <div class="no-results" id="noResults" style="display: none;">
            <div>🔍 No articles found matching your search.</div>
            <div style="font-size: 0.9rem; margin-top: 10px;">Try different keywords or clear your search.</div>
        </div>
        
        <div class="news-grid">
            {{range $source, $sourceInfo := .Sources}}
            {{$sourceItems := index $.GroupedItems $source}}
            {{if $sourceItems}}
            <div class="news-source">
                <div class="source-header">
                    <div class="source-icon" style="background-color: {{$sourceInfo.Color}};">{{$source}}</div>
                    <div class="source-name">{{$sourceInfo.Name}}</div>
                    <div class="updated-badge">Updated</div>
                    <div class="item-count">{{len $sourceItems}}</div>
                </div>
                <div class="news-items">
                    {{range $sourceItems}}
                    <div class="news-item {{if .HasNifty50}}nifty50-highlight{{end}}" data-nifty50="{{.HasNifty50}}" data-time="{{.TimeAgo}}" data-source="{{.Source}}">
                        {{if .HasNifty50}}
                        <span class="nifty50-badge" title="Mentions NIFTY50 stock: {{.Nifty50Stock}}">{{.Nifty50Stock}}</span>
                        {{end}}
                        <a href="{{.Link}}" class="news-title" target="_blank" rel="noopener noreferrer">{{.Title}}</a>
                        {{if .Description}}
                        <div class="news-description">{{.Description}}</div>
                        {{end}}
                        <div class="news-meta">
                            <span class="news-time">⏰ {{.TimeAgo}}</span>
                            {{if .Category}}
                            <span class="news-category">{{.Category}}</span>
                            {{else}}
                            <span class="news-category">General</span>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
            {{end}}
        </div>
    </div>
    
    <div class="action-buttons">
        <button class="action-btn scroll-top" id="scrollTop" title="Scroll to Top" style="display: none;">↑</button>
        <button class="action-btn" onclick="refreshNews()" title="Refresh News">🔄</button>
    </div>
    
    <script>
        // Global variables
        let allNewsItems = [];
        let filteredItems = [];
        let currentFilter = 'all';
        let searchTerm = '';
        let isDarkMode = localStorage.getItem('darkMode') === 'true';

        // Initialize the application
        document.addEventListener('DOMContentLoaded', function() {
            initializeTheme();
            collectNewsItems();
            updateCounts();
            initializeEventListeners();
            initializeScrollToTop();
            
            console.log('📈 Business News Aggregator loaded');
            console.log('🔄 Auto-refresh every 5 minutes');
        });

        // Theme Management
        function initializeTheme() {
            const themeToggle = document.getElementById('themeToggle');
            if (isDarkMode) {
                document.body.classList.add('dark');
                themeToggle.textContent = '☀️';
            } else {
                themeToggle.textContent = '🌙';
            }
        }

        function toggleTheme() {
            isDarkMode = !isDarkMode;
            document.body.classList.toggle('dark');
            const themeToggle = document.getElementById('themeToggle');
            themeToggle.textContent = isDarkMode ? '☀️' : '🌙';
            localStorage.setItem('darkMode', isDarkMode);
        }

        // Collect all news items for filtering and searching
        function collectNewsItems() {
            allNewsItems = Array.from(document.querySelectorAll('.news-item')).map(function(item) {
                return {
                    element: item,
                    title: item.querySelector('.news-title').textContent.toLowerCase(),
                    description: item.querySelector('.news-description') ? item.querySelector('.news-description').textContent.toLowerCase() : '',
                    isNifty50: item.dataset.nifty50 === 'true',
                    timeAgo: item.dataset.time,
                    source: item.dataset.source,
                    isRecent: isRecentArticle(item.dataset.time)
                };
            });
            filteredItems = allNewsItems.slice();
        }

        // Check if article is recent (within last 2 hours)
        function isRecentArticle(timeAgo) {
            if (timeAgo.includes('Just now') || timeAgo.includes('m ago')) {
                return true;
            }
            if (timeAgo.includes('h ago')) {
                const hours = parseInt(timeAgo.match(/\d+/)[0]);
                return hours <= 2;
            }
            return false;
        }

        // Update all counts
        function updateCounts() {
            const nifty50Count = allNewsItems.filter(function(item) { return item.isNifty50; }).length;
            const recentCount = allNewsItems.filter(function(item) { return item.isRecent; }).length;
            
            document.getElementById('allCount').textContent = allNewsItems.length;
            document.getElementById('niftyCount').textContent = nifty50Count;
            document.getElementById('recentCount').textContent = recentCount;
            document.getElementById('niftyMentions').textContent = nifty50Count;
            document.getElementById('totalCount').textContent = filteredItems.length;
        }

        // Search functionality
        function performSearch() {
            const query = searchTerm.toLowerCase();
            
            if (!query) {
                filteredItems = allNewsItems.filter(function(item) {
                    if (currentFilter === 'all') return true;
                    if (currentFilter === 'nifty50') return item.isNifty50;
                    if (currentFilter === 'recent') return item.isRecent;
                    return true;
                });
            } else {
                filteredItems = allNewsItems.filter(function(item) {
                    const matchesSearch = item.title.includes(query) || item.description.includes(query);
                    if (currentFilter === 'all') return matchesSearch;
                    if (currentFilter === 'nifty50') return matchesSearch && item.isNifty50;
                    if (currentFilter === 'recent') return matchesSearch && item.isRecent;
                    return matchesSearch;
                });
            }

            updateDisplay();
        }

        // Filter functionality
        function applyFilter(filter) {
            currentFilter = filter;
            
            // Update active filter button
            document.querySelectorAll('.filter-btn').forEach(function(btn) { btn.classList.remove('active'); });
            document.querySelector('[data-filter="' + filter + '"]').classList.add('active');
            
            performSearch(); // Re-apply search with new filter
        }

        // Update display based on filtered items
        function updateDisplay() {
            const newsGrid = document.querySelector('.news-grid');
            const noResults = document.getElementById('noResults');
            
            // Hide all news sources first
            document.querySelectorAll('.news-source').forEach(function(source) {
                source.style.display = 'none';
            });

            // Show items that match the filter
            let hasVisibleItems = false;
            const visibleSources = new Set();

            filteredItems.forEach(function(item) {
                const sourceContainer = item.element.closest('.news-source');
                visibleSources.add(sourceContainer);
                item.element.style.display = 'block';
                hasVisibleItems = true;
            });

            // Hide items that don't match
            allNewsItems.forEach(function(item) {
                if (!filteredItems.includes(item)) {
                    item.element.style.display = 'none';
                }
            });

            // Show source containers that have visible items
            visibleSources.forEach(function(source) {
                if (source) {
                    source.style.display = 'block';
                    
                    // Update item count for each source
                    const sourceItems = source.querySelectorAll('.news-item[style*="block"], .news-item:not([style*="none"])');
                    const countBadge = source.querySelector('.item-count');
                    if (countBadge) {
                        const visibleCount = Array.from(sourceItems).filter(function(item) {
                            return item.style.display !== 'none' && filteredItems.some(function(f) { return f.element === item; });
                        }).length;
                        countBadge.textContent = visibleCount;
                    }
                }
            });

            // Show/hide no results message
            noResults.style.display = hasVisibleItems ? 'none' : 'block';
            newsGrid.style.display = hasVisibleItems ? 'grid' : 'none';

            // Update total count
            document.getElementById('totalCount').textContent = filteredItems.length;
        }

        // Event listeners
        function initializeEventListeners() {
            const searchInput = document.getElementById('searchInput');
            const clearSearch = document.getElementById('clearSearch');
            const themeToggle = document.getElementById('themeToggle');

            // Search input
            searchInput.addEventListener('input', function(e) {
                searchTerm = e.target.value;
                clearSearch.style.display = searchTerm ? 'block' : 'none';
                performSearch();
            });

            // Clear search
            clearSearch.addEventListener('click', function() {
                searchInput.value = '';
                searchTerm = '';
                clearSearch.style.display = 'none';
                performSearch();
                searchInput.focus();
            });

            // Filter buttons
            document.querySelectorAll('.filter-btn').forEach(function(btn) {
                btn.addEventListener('click', function() {
                    applyFilter(this.dataset.filter);
                });
            });

            // Theme toggle
            themeToggle.addEventListener('click', toggleTheme);

            // Keyboard shortcuts
            document.addEventListener('keydown', function(e) {
                if (e.ctrlKey || e.metaKey) {
                    switch(e.key) {
                        case 'k':
                            e.preventDefault();
                            searchInput.focus();
                            break;
                        case 'd':
                            e.preventDefault();
                            toggleTheme();
                            break;
                        case 'r':
                            e.preventDefault();
                            refreshNews();
                            break;
                    }
                }
                if (e.key === 'Escape') {
                    if (document.activeElement === searchInput) {
                        searchInput.blur();
                    }
                }
            });
        }

        // Scroll to top functionality
        function initializeScrollToTop() {
            const scrollTopBtn = document.getElementById('scrollTop');
            
            window.addEventListener('scroll', function() {
                if (window.pageYOffset > 300) {
                    scrollTopBtn.style.display = 'flex';
                } else {
                    scrollTopBtn.style.display = 'none';
                }
            });

            scrollTopBtn.addEventListener('click', function() {
                window.scrollTo({
                    top: 0,
                    behavior: 'smooth'
                });
            });
        }

        // Enhanced refresh functionality
        function refreshNews() {
            const loadingContainer = document.getElementById('loadingContainer');
            const newsGrid = document.querySelector('.news-grid');
            
            // Show loading state
            loadingContainer.style.display = 'block';
            newsGrid.style.opacity = '0.5';
            
            // Add a small delay to show loading animation
            setTimeout(function() {
                location.reload();
            }, 500);
        }

        // Auto-refresh every 5 minutes (300 seconds)
        let refreshInterval = setInterval(function() {
            console.log('Auto-refreshing news...');
            refreshNews();
        }, 300000);
        
        // Update time ago every minute
        setInterval(function() {
            console.log('Updating time indicators...');
            // In a real app, this would update the time display
            // For now, we'll just log it as the page refreshes periodically
        }, 60000);
        
        // Show loading state on page unload
        window.addEventListener('beforeunload', function() {
            const loadingContainer = document.getElementById('loadingContainer');
            if (loadingContainer) {
                loadingContainer.style.display = 'block';
            }
        });

        // Performance monitoring
        window.addEventListener('load', function() {
            console.log('📊 Page loaded in', performance.now().toFixed(2), 'ms');
        });
    </script>
</body>
</html>
`

	// Group items by source
	groupedItems := make(map[string][]NewsItem)
	for _, item := range news {
		groupedItems[item.Source] = append(groupedItems[item.Source], item)
	}

	t := template.Must(template.New("home").Parse(tmpl))
	data := NewsData{
		Items:        news,
		GroupedItems: groupedItems,
		Sources:      rssSources,
		LastUpdated:  lastUpdated,
		TotalSources: len(rssSources),
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	news, lastUpdated := getNewsFromCache()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fmt.Fprintf(w, `{
		"items": %d,
		"last_updated": "%s",
		"sources": %d,
		"status": "success"
	}`, len(news), lastUpdated, len(rssSources))
}

func startPeriodicRefresh() {
	// Initial fetch
	fetchAllNews()

	// Set up periodic refresh every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			fetchAllNews()
		}
	}()
}

func main() {
	// Start the periodic refresh in the background
	go startPeriodicRefresh()

	// HTTP handlers
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/status", apiHandler)

	fmt.Println("🚀 RSS News Aggregator starting...")
	fmt.Println("📡 Fetching feeds from 8 sources:")
	for code, source := range rssSources {
		fmt.Printf("   • %s: %s\n", code, source.Name)
	}
	fmt.Println("🔄 Auto-refresh interval: 5 minutes")
	fmt.Println("🌐 Server running at http://localhost:8080")
	fmt.Println("📊 API status at http://localhost:8080/api/status")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
