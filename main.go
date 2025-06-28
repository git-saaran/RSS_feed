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

	// Try parsing with timezone first
	if t, err := time.Parse("02-Jan-2006 15:04:05 MST", dateStr); err == nil {
		return t
	}

	// Try parsing without timezone
	if t, err := time.Parse("02-Jan-2006 15:04:05", dateStr); err == nil {
		return t
	}

	// Try parsing without seconds
	if t, err := time.Parse("02-Jan-2006 15:04", dateStr); err == nil {
		return t
	}

	// Try other standard formats
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	// If all parsing fails, return current time but don't log common invalid dates
	if dateStr != "" && dateStr != "0000-00-00 00:00:00" {
		log.Printf("Failed to parse date: %s", dateStr)
	}
	return time.Now()
}

func timeAgo(t time.Time) string {
	duration := time.Since(t)

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

	return newsCache, lastCacheTime.Format("Jan 2, 2006 at 3:04 PM")
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
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 1600px;
            margin: 0 auto;
        }
        
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        
        .header h1 {
            color: white;
            font-size: 2.5rem;
            margin-bottom: 10px;
            text-shadow: 0 2px 4px rgba(0,0,0,0.3);
        }
        
        .header p {
            color: rgba(255,255,255,0.9);
            font-size: 1.1rem;
            margin-bottom: 5px;
        }
        
        .last-updated {
            color: rgba(255,255,255,0.8);
            font-size: 0.9rem;
            font-style: italic;
        }
        
        .news-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(380px, 1fr));
            gap: 20px;
        }
        
        .news-source {
            background: white;
            border-radius: 16px;
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
            overflow: hidden;
            backdrop-filter: blur(10px);
        }
        
        .source-header {
            padding: 16px 20px;
            display: flex;
            align-items: center;
            gap: 12px;
            border-bottom: 1px solid #e5e7eb;
        }
        
        .source-icon {
            width: 40px;
            height: 40px;
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: bold;
            font-size: 12px;
        }
        
        .source-name {
            font-weight: 600;
            color: #374151;
            flex: 1;
            font-size: 14px;
        }
        
        .updated-badge {
            background: #10b981;
            color: white;
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 500;
        }
        
        .item-count {
            background: #6b7280;
            color: white;
            padding: 2px 6px;
            border-radius: 10px;
            font-size: 11px;
            margin-left: 8px;
        }
        
        .news-items {
            max-height: 500px;
            overflow-y: auto;
        }
        
        .news-item {
            padding: 12px 20px;
            border-bottom: 1px solid #f3f4f6;
            transition: all 0.2s;
            position: relative;
            border-left: 3px solid transparent;
        }
        
        .nifty50-highlight {
            background-color: #fff9e6;
            border-left-color: #ffd700;
            box-shadow: 0 2px 8px rgba(255, 215, 0, 0.1);
        }
        
        .nifty50-badge {
            position: absolute;
            top: 8px;
            right: 15px;
            background: #ffd700;
            color: #000;
            padding: 2px 8px;
            border-radius: 10px;
            font-size: 10px;
            font-weight: bold;
            text-transform: uppercase;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
            z-index: 1;
        }
        
        .news-item:hover {
            background-color: #f9fafb;
        }
        
        .news-item:last-child {
            border-bottom: none;
        }
        
        .news-title {
            font-weight: 600;
            color: #1f2937;
            margin-bottom: 6px;
            line-height: 1.3;
            text-decoration: none;
            display: block;
            font-size: 14px;
        }
        
        .news-title:hover {
            color: #3b82f6;
        }
        
        .news-description {
            color: #6b7280;
            font-size: 12px;
            line-height: 1.4;
            margin-bottom: 8px;
        }
        
        .news-meta {
            display: flex;
            justify-content: space-between;
            align-items: center;
            font-size: 11px;
            color: #9ca3af;
        }
        
        .news-time {
            font-weight: 500;
        }
        
        .news-category {
            background: #eff6ff;
            color: #2563eb;
            padding: 2px 6px;
            border-radius: 8px;
            font-weight: 500;
        }
        
        .stats-bar {
            display: flex;
            justify-content: center;
            gap: 20px;
            margin-bottom: 20px;
            flex-wrap: wrap;
        }
        
        .stat-item {
            background: rgba(255,255,255,0.2);
            color: white;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 14px;
            backdrop-filter: blur(10px);
        }
        
        .refresh-btn {
            position: fixed;
            bottom: 30px;
            right: 30px;
            background: #3b82f6;
            color: white;
            border: none;
            border-radius: 50%;
            width: 60px;
            height: 60px;
            cursor: pointer;
            box-shadow: 0 4px 16px rgba(59, 130, 246, 0.3);
            transition: all 0.3s;
            font-size: 20px;
            z-index: 1000;
        }
        
        .refresh-btn:hover {
            background: #2563eb;
            transform: scale(1.1);
        }
        
        .loading {
            text-align: center;
            color: white;
            font-size: 1.2rem;
            margin: 50px 0;
        }
        
        @media (max-width: 768px) {
            .news-grid {
                grid-template-columns: 1fr;
            }
            
            .header h1 {
                font-size: 2rem;
            }
            
            .stats-bar {
                gap: 10px;
            }
            
            .stat-item {
                font-size: 12px;
                padding: 6px 12px;
            }
        }
        
        /* Scrollbar styling */
        .news-items::-webkit-scrollbar {
            width: 6px;
        }
        
        .news-items::-webkit-scrollbar-track {
            background: #f1f1f1;
        }
        
        .news-items::-webkit-scrollbar-thumb {
            background: #c1c1c1;
            border-radius: 3px;
        }
        
        .news-items::-webkit-scrollbar-thumb:hover {
            background: #a8a8a8;
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
        
        <div class="stats-bar">
            <div class="stat-item">📊 Total Articles: {{len .Items}}</div>
            <div class="stat-item">🔄 Auto-refresh: 5 min</div>
            <div class="stat-item">📡 Live Sources: {{.TotalSources}}</div>
        </div>
        
        <div class="news-grid">
            {{$sources := dict "TOI" "Times of India" "TH" "The Hindu" "BL" "Business Line" "LM" "LiveMint" "ZP" "Zerodha Pulse" "NSE_IT" "NSE Insider Trading" "NSE_BB" "NSE Buy Back" "NSE_FR" "NSE Financial Results"}}
            {{$sourceOrder := slice "BS_MARKETS" "BS_NEWS" "BS_COMMODITIES" "BS_IPO" "BS_STOCK_MARKET" "BS_CRYPTO" "TOI" "TH" "BL" "LM" "ZP" "NSE_IT" "NSE_BB" "NSE_FR"}}
            
            {{range $sourceOrder}}
            {{$source := .}}
            {{$sourceItems := where $.Items "Source" $source}}
            {{if $sourceItems}}
            <div class="news-source">
                <div class="source-header">
                    <div class="source-icon" style="background-color: {{(index $sourceItems 0).SourceColor}};">{{$source}}</div>
                    <div class="source-name">{{(index $sourceItems 0).SourceName}}</div>
                    <div class="updated-badge">Updated</div>
                    <div class="item-count">{{len $sourceItems}}</div>
                </div>
                <div class="news-items">
                    {{range $sourceItems}}
                    <div class="news-item {{if .HasNifty50}}nifty50-highlight{{end}}">
                        {{if .HasNifty50}}
                        <span class="nifty50-badge" title="Mentions NIFTY50 stock: {{.Nifty50Stock}}">{{.Nifty50Stock}}</span>
                        {{end}}
                        <a href="{{.Link}}" class="news-title" target="_blank" rel="noopener">{{.Title}}</a>
                        {{if .Description}}
                        <div class="news-description">{{.Description}}</div>
                        {{end}}
                        <div class="news-meta">
                            <span class="news-time">{{.TimeAgo}}</span>
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
    
    <button class="refresh-btn" onclick="location.reload()" title="Refresh News">🔄</button>
    
    <script>
        // Auto-refresh every 5 minutes (300 seconds)
        let refreshInterval = setInterval(function() {
            console.log('Auto-refreshing news...');
            location.reload();
        }, 300000);
        
        // Update time ago every minute
        setInterval(function() {
            // This would normally update the time ago text
            // For now, we'll just log it
            console.log('Updating time indicators...');
        }, 60000);
        
        // Show loading state on refresh
        window.addEventListener('beforeunload', function() {
            document.body.innerHTML = '<div class="loading">🔄 Refreshing news...</div>';
        });
        
        console.log('📈 Business News Aggregator loaded');
        console.log('🔄 Auto-refresh every 5 minutes');
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
