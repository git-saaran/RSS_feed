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
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" rel="stylesheet">
    <style>
        :root {
            --primary-gradient: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            --dark-gradient: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            --card-bg: rgba(255, 255, 255, 0.95);
            --card-bg-dark: rgba(30, 30, 46, 0.95);
            --text-primary: #1a202c;
            --text-primary-dark: #e2e8f0;
            --text-secondary: #4a5568;
            --text-secondary-dark: #a0aec0;
            --accent-color: #4f46e5;
            --success-color: #10b981;
            --warning-color: #f59e0b;
            --error-color: #ef4444;
            --border-color: rgba(0, 0, 0, 0.1);
            --border-color-dark: rgba(255, 255, 255, 0.1);
            --shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.1);
            --shadow-md: 0 4px 16px rgba(0, 0, 0, 0.1);
            --shadow-lg: 0 10px 40px rgba(0, 0, 0, 0.15);
            --border-radius: 16px;
            --transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        }
        
        [data-theme="dark"] {
            --card-bg: var(--card-bg-dark);
            --text-primary: var(--text-primary-dark);
            --text-secondary: var(--text-secondary-dark);
            --border-color: var(--border-color-dark);
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--primary-gradient);
            min-height: 100vh;
            padding: 20px;
            color: var(--text-primary);
            transition: var(--transition);
            overflow-x: hidden;
        }
        
        [data-theme="dark"] body {
            background: var(--dark-gradient);
        }
        
        .container {
            max-width: 1600px;
            margin: 0 auto;
            animation: slideUp 0.8s ease-out;
        }
        
        @keyframes slideUp {
            from {
                opacity: 0;
                transform: translateY(30px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }
        
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.7; }
        }
        
        @keyframes shimmer {
            0% { background-position: -200px 0; }
            100% { background-position: calc(200px + 100%) 0; }
        }
        
        .header {
            text-align: center;
            margin-bottom: 40px;
            position: relative;
        }
        
        .header::before {
            content: '';
            position: absolute;
            top: -10px;
            left: 50%;
            transform: translateX(-50%);
            width: 100px;
            height: 4px;
            background: linear-gradient(90deg, var(--accent-color), var(--success-color));
            border-radius: 2px;
            animation: pulse 2s infinite;
        }
        
        .header h1 {
            color: white;
            font-size: clamp(2rem, 4vw, 3rem);
            font-weight: 700;
            margin-bottom: 16px;
            text-shadow: 0 4px 8px rgba(0,0,0,0.3);
            letter-spacing: -0.02em;
        }
        
        .header p {
            color: rgba(255,255,255,0.9);
            font-size: clamp(1rem, 2vw, 1.2rem);
            margin-bottom: 8px;
            font-weight: 400;
        }
        
        .last-updated {
            color: rgba(255,255,255,0.8);
            font-size: 0.9rem;
            font-style: italic;
            font-family: 'JetBrains Mono', monospace;
            background: rgba(255,255,255,0.1);
            padding: 8px 16px;
            border-radius: 20px;
            display: inline-block;
            backdrop-filter: blur(10px);
            margin-top: 8px;
        }
        
        .controls {
            display: flex;
            justify-content: center;
            align-items: center;
            gap: 16px;
            margin-bottom: 30px;
            flex-wrap: wrap;
        }
        
        .theme-toggle {
            background: rgba(255,255,255,0.2);
            border: 1px solid rgba(255,255,255,0.3);
            color: white;
            padding: 10px 16px;
            border-radius: 25px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
            transition: var(--transition);
            backdrop-filter: blur(10px);
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .theme-toggle:hover {
            background: rgba(255,255,255,0.3);
            transform: translateY(-2px);
        }
        
        .search-box {
            position: relative;
            width: 300px;
            max-width: 100%;
        }
        
        .search-input {
            width: 100%;
            padding: 12px 40px 12px 16px;
            border: 1px solid rgba(255,255,255,0.3);
            border-radius: 25px;
            background: rgba(255,255,255,0.2);
            color: white;
            font-size: 14px;
            backdrop-filter: blur(10px);
            transition: var(--transition);
        }
        
        .search-input::placeholder {
            color: rgba(255,255,255,0.7);
        }
        
        .search-input:focus {
            outline: none;
            background: rgba(255,255,255,0.3);
            border-color: rgba(255,255,255,0.5);
        }
        
        .search-icon {
            position: absolute;
            right: 14px;
            top: 50%;
            transform: translateY(-50%);
            color: rgba(255,255,255,0.7);
        }
        
        .stats-bar {
            display: flex;
            justify-content: center;
            gap: 20px;
            margin-bottom: 30px;
            flex-wrap: wrap;
        }
        
        .stat-item {
            background: rgba(255,255,255,0.15);
            color: white;
            padding: 12px 20px;
            border-radius: 20px;
            font-size: 14px;
            font-weight: 500;
            backdrop-filter: blur(15px);
            border: 1px solid rgba(255,255,255,0.2);
            transition: var(--transition);
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .stat-item:hover {
            transform: translateY(-2px);
            background: rgba(255,255,255,0.25);
        }
        
        .stat-icon {
            font-size: 16px;
        }
        
        .news-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(380px, 1fr));
            gap: 24px;
            animation: fadeIn 1s ease-out 0.2s both;
        }
        
        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }
        
        .news-source {
            background: var(--card-bg);
            border-radius: var(--border-radius);
            box-shadow: var(--shadow-lg);
            overflow: hidden;
            backdrop-filter: blur(20px);
            border: 1px solid var(--border-color);
            transition: var(--transition);
            position: relative;
            animation: slideUp 0.6s ease-out;
        }
        
        .news-source:hover {
            transform: translateY(-4px);
            box-shadow: 0 20px 60px rgba(0,0,0,0.2);
        }
        
        .source-header {
            padding: 20px 24px;
            display: flex;
            align-items: center;
            gap: 16px;
            border-bottom: 1px solid var(--border-color);
            background: linear-gradient(135deg, rgba(255,255,255,0.1), rgba(255,255,255,0.05));
            position: relative;
            overflow: hidden;
        }
        
        .source-header::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: linear-gradient(90deg, transparent, rgba(255,255,255,0.1), transparent);
            transition: left 0.5s;
        }
        
        .news-source:hover .source-header::before {
            left: 100%;
        }
        
        .source-icon {
            width: 48px;
            height: 48px;
            border-radius: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: 700;
            font-size: 12px;
            position: relative;
            overflow: hidden;
            box-shadow: var(--shadow-md);
        }
        
        .source-icon::before {
            content: '';
            position: absolute;
            top: -50%;
            left: -50%;
            width: 200%;
            height: 200%;
            background: linear-gradient(45deg, transparent, rgba(255,255,255,0.2), transparent);
            transition: transform 0.5s;
            transform: rotate(45deg) translateX(-100%);
        }
        
        .news-source:hover .source-icon::before {
            transform: rotate(45deg) translateX(100%);
        }
        
        .source-name {
            font-weight: 600;
            color: var(--text-primary);
            flex: 1;
            font-size: 16px;
            letter-spacing: -0.01em;
        }
        
        .source-badges {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .updated-badge {
            background: linear-gradient(135deg, var(--success-color), #059669);
            color: white;
            padding: 6px 12px;
            border-radius: 16px;
            font-size: 12px;
            font-weight: 600;
            box-shadow: var(--shadow-sm);
            animation: pulse 2s infinite;
        }
        
        .item-count {
            background: linear-gradient(135deg, var(--accent-color), #3730a3);
            color: white;
            padding: 4px 10px;
            border-radius: 12px;
            font-size: 11px;
            font-weight: 600;
            font-family: 'JetBrains Mono', monospace;
            box-shadow: var(--shadow-sm);
        }
        
        .news-items {
            max-height: 520px;
            overflow-y: auto;
            scroll-behavior: smooth;
        }
        
        .news-item {
            padding: 16px 24px;
            border-bottom: 1px solid var(--border-color);
            transition: var(--transition);
            position: relative;
            border-left: 3px solid transparent;
        }
        
        .news-item::before {
            content: '';
            position: absolute;
            left: 0;
            top: 0;
            width: 0;
            height: 100%;
            background: linear-gradient(135deg, var(--accent-color), var(--success-color));
            transition: width 0.3s ease;
        }
        
        .news-item:hover::before {
            width: 3px;
        }
        
        .nifty50-highlight {
            background: linear-gradient(135deg, #fef3c7, #fde68a);
            border-left-color: var(--warning-color);
            position: relative;
        }
        
        [data-theme="dark"] .nifty50-highlight {
            background: linear-gradient(135deg, rgba(245, 158, 11, 0.1), rgba(245, 158, 11, 0.05));
        }
        
        .nifty50-badge {
            position: absolute;
            top: 12px;
            right: 20px;
            background: linear-gradient(135deg, var(--warning-color), #d97706);
            color: white;
            padding: 4px 10px;
            border-radius: 12px;
            font-size: 10px;
            font-weight: 700;
            text-transform: uppercase;
            box-shadow: var(--shadow-md);
            z-index: 1;
            animation: pulse 3s infinite;
        }
        
        .news-item:hover {
            background: rgba(79, 70, 229, 0.03);
            transform: translateX(4px);
        }
        
        [data-theme="dark"] .news-item:hover {
            background: rgba(79, 70, 229, 0.1);
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
            letter-spacing: -0.01em;
            transition: var(--transition);
        }
        
        .news-title:hover {
            color: var(--accent-color);
            text-decoration: underline;
        }
        
        .news-description {
            color: var(--text-secondary);
            font-size: 13px;
            line-height: 1.5;
            margin-bottom: 12px;
            font-weight: 400;
        }
        
        .news-meta {
            display: flex;
            justify-content: space-between;
            align-items: center;
            font-size: 12px;
            color: var(--text-secondary);
        }
        
        .news-time {
            font-weight: 500;
            font-family: 'JetBrains Mono', monospace;
            display: flex;
            align-items: center;
            gap: 4px;
        }
        
        .news-category {
            background: linear-gradient(135deg, #eff6ff, #dbeafe);
            color: var(--accent-color);
            padding: 4px 10px;
            border-radius: 12px;
            font-weight: 600;
            font-size: 11px;
            box-shadow: var(--shadow-sm);
            border: 1px solid rgba(79, 70, 229, 0.1);
        }
        
        [data-theme="dark"] .news-category {
            background: rgba(79, 70, 229, 0.2);
            color: #a5b4fc;
            border-color: rgba(79, 70, 229, 0.3);
        }
        
        .floating-controls {
            position: fixed;
            bottom: 30px;
            right: 30px;
            display: flex;
            flex-direction: column;
            gap: 12px;
            z-index: 1000;
        }
        
        .control-btn {
            width: 56px;
            height: 56px;
            border-radius: 50%;
            border: none;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 20px;
            font-weight: 600;
            transition: var(--transition);
            box-shadow: var(--shadow-lg);
            backdrop-filter: blur(20px);
        }
        
        .refresh-btn {
            background: linear-gradient(135deg, var(--accent-color), #3730a3);
            color: white;
        }
        
        .refresh-btn:hover {
            transform: scale(1.1) rotate(180deg);
            box-shadow: 0 8px 32px rgba(79, 70, 229, 0.4);
        }
        
        .scroll-top-btn {
            background: linear-gradient(135deg, var(--success-color), #059669);
            color: white;
            opacity: 0;
            visibility: hidden;
        }
        
        .scroll-top-btn.visible {
            opacity: 1;
            visibility: visible;
        }
        
        .scroll-top-btn:hover {
            transform: scale(1.1);
            box-shadow: 0 8px 32px rgba(16, 185, 129, 0.4);
        }
        
        .loading-overlay {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.8);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 9999;
            opacity: 0;
            visibility: hidden;
            transition: var(--transition);
        }
        
        .loading-overlay.show {
            opacity: 1;
            visibility: visible;
        }
        
        .loading-spinner {
            width: 60px;
            height: 60px;
            border: 4px solid rgba(255, 255, 255, 0.3);
            border-left: 4px solid white;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        .loading-text {
            color: white;
            font-size: 18px;
            font-weight: 500;
            margin-top: 20px;
        }
        
        /* Scrollbar styling */
        .news-items::-webkit-scrollbar {
            width: 8px;
        }
        
        .news-items::-webkit-scrollbar-track {
            background: rgba(0, 0, 0, 0.05);
            border-radius: 4px;
        }
        
        .news-items::-webkit-scrollbar-thumb {
            background: linear-gradient(135deg, var(--accent-color), var(--success-color));
            border-radius: 4px;
            transition: var(--transition);
        }
        
        .news-items::-webkit-scrollbar-thumb:hover {
            background: linear-gradient(135deg, #3730a3, #059669);
        }
        
        /* Mobile optimizations */
        @media (max-width: 768px) {
            body {
                padding: 15px;
            }
            
            .news-grid {
                grid-template-columns: 1fr;
                gap: 20px;
            }
            
            .stats-bar {
                gap: 12px;
            }
            
            .stat-item {
                font-size: 12px;
                padding: 8px 14px;
            }
            
            .controls {
                flex-direction: column;
                gap: 12px;
            }
            
            .search-box {
                width: 100%;
                max-width: 300px;
            }
            
            .floating-controls {
                bottom: 20px;
                right: 20px;
            }
            
            .control-btn {
                width: 50px;
                height: 50px;
                font-size: 18px;
            }
            
            .news-item {
                padding: 14px 18px;
            }
            
            .source-header {
                padding: 16px 18px;
            }
        }
        
        /* Print styles */
        @media print {
            body {
                background: white !important;
                color: black !important;
            }
            
            .floating-controls,
            .controls,
            .stats-bar {
                display: none !important;
            }
            
            .news-source {
                break-inside: avoid;
                box-shadow: none !important;
                border: 1px solid #ccc !important;
            }
        }
        
        /* Accessibility improvements */
        @media (prefers-reduced-motion: reduce) {
            *,
            *::before,
            *::after {
                animation-duration: 0.01ms !important;
                animation-iteration-count: 1 !important;
                transition-duration: 0.01ms !important;
            }
        }
        
        /* Focus styles for better keyboard navigation */
        .news-title:focus,
        .control-btn:focus,
        .theme-toggle:focus,
        .search-input:focus {
            outline: 2px solid var(--accent-color);
            outline-offset: 2px;
        }
    </style>
</head>
<body>
    <div class="loading-overlay" id="loadingOverlay">
        <div style="text-align: center;">
            <div class="loading-spinner"></div>
            <div class="loading-text">🔄 Refreshing news...</div>
        </div>
    </div>
    
    <div class="container">
        <div class="header">
            <h1><i class="fas fa-chart-line"></i> Business News Aggregator</h1>
            <p>Real-time updates from {{.TotalSources}} premium financial sources</p>
            <div class="last-updated">
                <i class="far fa-clock"></i> Last updated: {{.LastUpdated}}
            </div>
        </div>
        
        <div class="controls">
            <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
                <i class="fas fa-moon" id="themeIcon"></i>
                <span id="themeText">Dark Mode</span>
            </button>
            <div class="search-box">
                <input type="text" class="search-input" placeholder="Search news..." id="searchInput">
                <i class="fas fa-search search-icon"></i>
            </div>
        </div>
        
        <div class="stats-bar">
            <div class="stat-item">
                <i class="fas fa-newspaper stat-icon"></i>
                <span>{{len .Items}} Articles</span>
            </div>
            <div class="stat-item">
                <i class="fas fa-sync-alt stat-icon"></i>
                <span>Auto-refresh: 5 min</span>
            </div>
            <div class="stat-item">
                <i class="fas fa-broadcast-tower stat-icon"></i>
                <span>{{.TotalSources}} Live Sources</span>
            </div>
            <div class="stat-item">
                <i class="fas fa-chart-line stat-icon"></i>
                <span id="niftyCount">0 NIFTY50 mentions</span>
            </div>
        </div>
        
        <div class="news-grid" id="newsGrid">
            {{$sources := dict "TOI" "Times of India" "TH" "The Hindu" "BL" "Business Line" "LM" "LiveMint" "ZP" "Zerodha Pulse" "NSE_IT" "NSE Insider Trading" "NSE_BB" "NSE Buy Back" "NSE_FR" "NSE Financial Results" "NDTV_PROFIT" "NDTV Profit"}}
            {{$sourceOrder := slice "BS_MARKETS" "BS_NEWS" "BS_COMMODITIES" "BS_IPO" "BS_STOCK_MARKET" "BS_CRYPTO" "NDTV_PROFIT" "TOI" "TH" "BL" "LM" "ZP" "NSE_IT" "NSE_BB" "NSE_FR"}}
            
            {{range $sourceOrder}}
            {{$source := .}}
            {{$sourceItems := where $.Items "Source" $source}}
            {{if $sourceItems}}
            <div class="news-source" data-source="{{$source}}">
                <div class="source-header">
                    <div class="source-icon" style="background: linear-gradient(135deg, {{(index $sourceItems 0).SourceColor}}, {{(index $sourceItems 0).SourceColor}}dd);">
                        {{$source}}
                    </div>
                    <div class="source-name">{{(index $sourceItems 0).SourceName}}</div>
                    <div class="source-badges">
                        <div class="updated-badge">
                            <i class="fas fa-check-circle"></i> Updated
                        </div>
                        <div class="item-count">{{len $sourceItems}}</div>
                    </div>
                </div>
                <div class="news-items">
                    {{range $sourceItems}}
                    <div class="news-item {{if .HasNifty50}}nifty50-highlight{{end}}" data-title="{{.Title | lower}}" data-description="{{.Description | lower}}">
                        {{if .HasNifty50}}
                        <span class="nifty50-badge" title="Mentions NIFTY50 stock: {{.Nifty50Stock}}">
                            <i class="fas fa-star"></i> {{.Nifty50Stock}}
                        </span>
                        {{end}}
                        <a href="{{.Link}}" class="news-title" target="_blank" rel="noopener">{{.Title}}</a>
                        {{if .Description}}
                        <div class="news-description">{{.Description}}</div>
                        {{end}}
                        <div class="news-meta">
                            <span class="news-time">
                                <i class="far fa-clock"></i> {{.TimeAgo}}
                            </span>
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
    
    <div class="floating-controls">
        <button class="control-btn scroll-top-btn" onclick="scrollToTop()" title="Scroll to top" aria-label="Scroll to top">
            <i class="fas fa-chevron-up"></i>
        </button>
        <button class="control-btn refresh-btn" onclick="refreshNews()" title="Refresh news" aria-label="Refresh news">
            <i class="fas fa-sync-alt"></i>
        </button>
    </div>
    
    <script>
        // Theme management
        let isDarkMode = localStorage.getItem('darkMode') === 'true';
        
        function initTheme() {
            if (isDarkMode) {
                document.documentElement.setAttribute('data-theme', 'dark');
                document.getElementById('themeIcon').className = 'fas fa-sun';
                document.getElementById('themeText').textContent = 'Light Mode';
            }
        }
        
        function toggleTheme() {
            isDarkMode = !isDarkMode;
            localStorage.setItem('darkMode', isDarkMode);
            
            if (isDarkMode) {
                document.documentElement.setAttribute('data-theme', 'dark');
                document.getElementById('themeIcon').className = 'fas fa-sun';
                document.getElementById('themeText').textContent = 'Light Mode';
            } else {
                document.documentElement.removeAttribute('data-theme');
                document.getElementById('themeIcon').className = 'fas fa-moon';
                document.getElementById('themeText').textContent = 'Dark Mode';
            }
        }
        
        // Search functionality
        const searchInput = document.getElementById('searchInput');
        const newsGrid = document.getElementById('newsGrid');
        
        searchInput.addEventListener('input', function() {
            const query = this.value.toLowerCase().trim();
            const newsSources = newsGrid.querySelectorAll('.news-source');
            
            newsSources.forEach(source => {
                const newsItems = source.querySelectorAll('.news-item');
                let visibleItems = 0;
                
                newsItems.forEach(item => {
                    const title = item.getAttribute('data-title') || '';
                    const description = item.getAttribute('data-description') || '';
                    
                    if (query === '' || title.includes(query) || description.includes(query)) {
                        item.style.display = 'block';
                        visibleItems++;
                    } else {
                        item.style.display = 'none';
                    }
                });
                
                // Hide source if no items are visible
                source.style.display = visibleItems > 0 ? 'block' : 'none';
            });
        });
        
        // Scroll to top functionality
        const scrollTopBtn = document.querySelector('.scroll-top-btn');
        
        window.addEventListener('scroll', function() {
            if (window.pageYOffset > 300) {
                scrollTopBtn.classList.add('visible');
            } else {
                scrollTopBtn.classList.remove('visible');
            }
        });
        
        function scrollToTop() {
            window.scrollTo({
                top: 0,
                behavior: 'smooth'
            });
        }
        
        // Refresh functionality
        function refreshNews() {
            const loadingOverlay = document.getElementById('loadingOverlay');
            loadingOverlay.classList.add('show');
            
            setTimeout(() => {
                location.reload();
            }, 500);
        }
        
        // Count NIFTY50 mentions
        function countNiftyMentions() {
            const niftyItems = document.querySelectorAll('.nifty50-highlight');
            const count = niftyItems.length;
            document.getElementById('niftyCount').textContent = count + ' NIFTY50 mentions';
        }
        
        // Auto-refresh functionality
        let refreshInterval = setInterval(function() {
            console.log('Auto-refreshing news...');
            refreshNews();
        }, 300000); // 5 minutes
        
        // Update time indicators every minute
        setInterval(function() {
            console.log('Time indicators updated');
        }, 60000);
        
        // Keyboard shortcuts
        document.addEventListener('keydown', function(e) {
            // Ctrl/Cmd + R for refresh
            if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
                e.preventDefault();
                refreshNews();
            }
            
            // Ctrl/Cmd + D for dark mode
            if ((e.ctrlKey || e.metaKey) && e.key === 'd') {
                e.preventDefault();
                toggleTheme();
            }
            
            // Escape to clear search
            if (e.key === 'Escape') {
                searchInput.value = '';
                searchInput.dispatchEvent(new Event('input'));
            }
        });
        
        // Initialize on page load
        document.addEventListener('DOMContentLoaded', function() {
            initTheme();
            countNiftyMentions();
            console.log('📈 Enhanced Business News Aggregator loaded');
            console.log('🔄 Auto-refresh every 5 minutes');
            console.log('⌨️  Keyboard shortcuts: Ctrl+R (refresh), Ctrl+D (theme), Esc (clear search)');
        });
        
        // Performance optimization - lazy loading for images if any
        if ('IntersectionObserver' in window) {
            const imageObserver = new IntersectionObserver((entries, observer) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        const img = entry.target;
                        img.src = img.dataset.src;
                        img.classList.remove('lazy');
                        imageObserver.unobserve(img);
                    }
                });
            });
        }
        
        // Add smooth scrolling for better UX
        document.documentElement.style.scrollBehavior = 'smooth';
        
        // Add focus management for accessibility
        searchInput.addEventListener('focus', function() {
            this.style.transform = 'scale(1.02)';
        });
        
        searchInput.addEventListener('blur', function() {
            this.style.transform = 'scale(1)';
        });
    </script>
</body>
</html>
`

	// Template helper functions
	funcMap := template.FuncMap{
		"dict": func(values ...interface{}) map[string]interface{} {
			dict := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				key := values[i].(string)
				value := values[i+1]
				dict[key] = value
			}
			return dict
		},
		"slice": func(values ...string) []string {
			return values
		},
		"where": func(items []NewsItem, field, value string) []NewsItem {
			var result []NewsItem
			for _, item := range items {
				switch field {
				case "Source":
					if item.Source == value {
						result = append(result, item)
					}
				}
			}
			return result
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
	}

	t := template.Must(template.New("home").Funcs(funcMap).Parse(tmpl))
	data := NewsData{
		Items:        news,
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
