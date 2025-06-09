package main

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// GenerateID creates a unique ID for news items
func GenerateID(guid, link, title string) string {
	var source string
	if guid != "" {
		source = guid
	} else if link != "" {
		source = link
	} else {
		source = title + fmt.Sprintf("%d", time.Now().Unix())
	}
	
	hash := md5.Sum([]byte(source))
	return fmt.Sprintf("%x", hash)[:16]
}

// ExtractStockSymbols finds stock symbols in text
func ExtractStockSymbols(text string) []string {
	upperText := strings.ToUpper(text)
	var symbols []string
	symbolMap := make(map[string]bool)
	
	// Check against all stock keyword categories
	for category, keywords := range StockKeywords {
		for _, keyword := range keywords {
			if strings.Contains(upperText, keyword) && !symbolMap[keyword] {
				symbols = append(symbols, keyword)
				symbolMap[keyword] = true
				
				// Limit to prevent spam
				if len(symbols) >= 10 {
					break
				}
			}
		}
		if len(symbols) >= 10 {
			break
		}
	}
	
	// Sort symbols for consistency
	sort.Strings(symbols)
	return symbols
}

// AnalyzeSentiment determines sentiment from text
func AnalyzeSentiment(text string) string {
	upperText := strings.ToUpper(text)
	
	positiveScore := 0
	negativeScore := 0
	neutralScore := 0
	
	// Count sentiment keywords
	for _, keyword := range SentimentKeywords["positive"] {
		if strings.Contains(upperText, strings.ToUpper(keyword)) {
			positiveScore++
		}
	}
	
	for _, keyword := range SentimentKeywords["negative"] {
		if strings.Contains(upperText, strings.ToUpper(keyword)) {
			negativeScore++
		}
	}
	
	for _, keyword := range SentimentKeywords["neutral"] {
		if strings.Contains(upperText, strings.ToUpper(keyword)) {
			neutralScore++
		}
	}
	
	// Determine dominant sentiment
	if positiveScore > negativeScore && positiveScore > neutralScore {
		return "positive"
	} else if negativeScore > positiveScore && negativeScore > neutralScore {
		return "negative"
	}
	return "neutral"
}

// CleanText removes HTML tags and decodes entities
func CleanText(text string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(text, "")
	
	// Decode HTML entities
	entities := map[string]string{
		"&amp;":     "&",
		"&lt;":      "<",
		"&gt;":      ">",
		"&quot;":    "\"",
		"&#39;":     "'",
		"&apos;":    "'",
		"&nbsp;":    " ",
		"&hellip;":  "...",
		"&mdash;":   "—",
		"&ndash;":   "–",
		"&rsquo;":   "'",
		"&lsquo;":   "'",
		"&rdquo;":   """,
		"&ldquo;":   """,
		"&copy;":    "©",
		"&reg;":     "®",
		"&trade;":   "™",
		"&bull;":    "•",
		"&laquo;":   "«",
		"&raquo;":   "»",
	}
	
	for entity, replacement := range entities {
		cleaned = strings.ReplaceAll(cleaned, entity, replacement)
	}
	
	// Clean up whitespace
	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	
	// Remove extra punctuation
	cleaned = regexp.MustCompile(`[.]{3,}`).ReplaceAllString(cleaned, "...")
	
	return cleaned
}

// ParseDate parses various date formats
func ParseDate(dateStr string) time.Time {
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC822Z,
		time.RFC822,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"Jan 2, 2006 3:04:05 PM",
		"January 2, 2006 3:04:05 PM",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02-01-2006 15:04:05",
	}
	
	dateStr = strings.TrimSpace(dateStr)
	
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}
	
	return time.Now()
}

// GetTimeAgo returns human-readable time difference
func GetTimeAgo(t time.Time) string {
	diff := time.Since(t)
	
	switch {
	case diff < time.Minute:
		return "Just now"
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// RemoveDuplicates removes duplicate news items
func RemoveDuplicates(news []NewsItem) []NewsItem {
	seen := make(map[string]bool)
	var unique []NewsItem
	
	for _, item := range news {
		if !seen[item.ID] {
			seen[item.ID] = true
			unique = append(unique, item)
		}
	}
	
	return unique
}

// CountWords counts words in text
func CountWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

// CalculateReadTime estimates reading time in minutes
func CalculateReadTime(text string) int {
	wordCount := CountWords(text)
	// Average reading speed: 200-250 words per minute
	readTime := wordCount / 225
	if readTime < 1 {
		return 1
	}
	return readTime
}

// CalculateScore calculates news item score for ranking
func CalculateScore(title, description string, stockSymbols []string, sourcePriority int) float64 {
	score := 0.0
	
	// Base score from source priority (higher priority = higher score)
	score += float64(5 - sourcePriority) // Priority 1 gets 4 points, priority 5 gets 0
	
	// Stock relevance bonus
	score += float64(len(stockSymbols)) * 0.5
	
	// Title length bonus (optimal around 60-80 characters)
	titleLen := len(title)
	if titleLen >= 40 && titleLen <= 100 {
		score += 1.0
	} else if titleLen >= 20 && titleLen <= 150 {
		score += 0.5
	}
	
	// Description quality bonus
	descLen := len(description)
	if descLen >= 100 && descLen <= 500 {
		score += 1.0
	} else if descLen >= 50 {
		score += 0.5
	}
	
	// Sentiment bonus (neutral gets slight bonus for balance)
	sentiment := AnalyzeSentiment(title + " " + description)
	switch sentiment {
	case "positive", "negative":
		score += 0.3
	case "neutral":
		score += 0.1
	}
	
	// Keywords bonus
	text := strings.ToUpper(title + " " + description)
	keywordCount := 0
	for _, keywords := range StockKeywords {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				keywordCount++
				if keywordCount >= 5 {
					break
				}
			}
		}
		if keywordCount >= 5 {
			break
		}
	}
	score += float64(keywordCount) * 0.1
	
	return score
}

// ExtractTags extracts relevant tags from news content
func ExtractTags(title, description string) []string {
	text := strings.ToUpper(title + " " + description)
	var tags []string
	tagMap := make(map[string]bool)
	
	// Add category tags based on keywords
	categories := map[string][]string{
		"banking":      {"BANK", "SBI", "HDFC", "ICICI", "AXIS", "KOTAK"},
		"technology":   {"IT", "TECH", "TCS", "INFOSYS", "WIPRO", "SOFTWARE"},
		"energy":       {"OIL", "GAS", "ENERGY", "RELIANCE", "ONGC", "BPCL"},
		"automotive":   {"AUTO", "CAR", "MARUTI", "TATA MOTORS", "BAJAJ"},
		"pharma":       {"PHARMA", "DRUG", "MEDICINE", "SUN PHARMA", "DR REDDY"},
		"fmcg":         {"FMCG", "CONSUMER", "UNILEVER", "ITC", "NESTLE"},
		"metals":       {"STEEL", "METAL", "TATA STEEL", "JSW", "HINDALCO"},
		"telecom":      {"TELECOM", "MOBILE", "BHARTI", "AIRTEL", "JIO"},
		"realty":       {"REAL ESTATE", "PROPERTY", "REALTY", "DLF"},
		"mutual_fund":  {"MUTUAL FUND", "MF", "SIP", "NAV"},
		"ipo":          {"IPO", "LISTING", "PUBLIC OFFERING"},
		"results":      {"RESULTS", "EARNINGS", "QUARTERLY", "ANNUAL"},
		"dividend":     {"DIVIDEND", "PAYOUT", "YIELD"},
		"merger":       {"MERGER", "ACQUISITION", "DEAL", "TAKEOVER"},
	}
	
	for tag, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) && !tagMap[tag] {
				tags = append(tags, tag)
				tagMap[tag] = true
				break
			}
		}
	}
	
	// Add sentiment tag
	sentiment := AnalyzeSentiment(text)
	if sentiment != "neutral" && !tagMap[sentiment] {
		tags = append(tags, sentiment)
		tagMap[sentiment] = true
	}
	
	// Limit tags
	if len(tags) > 5 {
		tags = tags[:5]
	}
	
	return tags
}

// FilterNews filters news based on given criteria
func FilterNews(news []NewsItem, filter FilterOptions) ([]NewsItem, int) {
	var filtered []NewsItem
	
	for _, item := range news {
		// Source filter
		if filter.Source != "" && item.Source != filter.Source {
			continue
		}
		
		// Category filter
		if filter.Category != "" && item.Category != filter.Category {
			continue
		}
		
		// Sentiment filter
		if filter.Sentiment != "" && item.Sentiment != filter.Sentiment {
			continue
		}
		
		// Stock news filter
		if filter.StockOnly && !item.IsStockNews {
			continue
		}
		
		// Date range filter
		if !filter.DateFrom.IsZero() && item.PubDate.Before(filter.DateFrom) {
			continue
		}
		if !filter.DateTo.IsZero() &&ī