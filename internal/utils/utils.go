package utils

import (
	"crypto/md5"
	"fmt"
	"regexp"
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

	for _, keywords := range StockKeywords {
		for _, keyword := range keywords {
			if strings.Contains(upperText, keyword) && !symbolMap[keyword] {
				symbols = append(symbols, keyword)
				symbolMap[keyword] = true
				if len(symbols) >= 10 {
					break
				}
			}
		}
		if len(symbols) >= 10 {
			break
		}
	}

	sort.Strings(symbols)
	return symbols
}

// AnalyzeSentiment determines sentiment from text
func AnalyzeSentiment(text string) string {
	upperText := strings.ToUpper(text)

	positiveScore, negativeScore, neutralScore := 0, 0, 0

	for _, keyword := range SentimentKeywords["positive"] {
		if strings.Contains(upperText, keyword) {
			positiveScore++
		}
	}
	for _, keyword := range SentimentKeywords["negative"] {
		if strings.Contains(upperText, keyword) {
			negativeScore++
		}
	}
	for _, keyword := range SentimentKeywords["neutral"] {
		if strings.Contains(upperText, keyword) {
			neutralScore++
		}
	}

	if positiveScore > negativeScore && positiveScore > neutralScore {
		return "positive"
	} else if negativeScore > positiveScore && negativeScore > neutralScore {
		return "negative"
	}
	return "neutral"
}

// CleanText removes HTML tags and decodes entities
func CleanText(text string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(text, "")

	entities := map[string]string{
		"&amp;": "&", "&lt;": "<", "&gt;": ">", "&quot;": "\"", "&#39;": "'",
		"&apos;": "'", "&nbsp;": " ", "&hellip;": "...", "&mdash;": "—",
		"&ndash;": "–", "&rsquo;": "'", "&lsquo;": "'", "&rdquo;": "\"",
		"&ldquo;": "\"", "&copy;": "©", "&reg;": "®", "&trade;": "™",
		"&bull;": "•", "&laquo;": "«", "&raquo;": "»",
	}

	for entity, replacement := range entities {
		cleaned = strings.ReplaceAll(cleaned, entity, replacement)
	}

	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`[.]{3,}`).ReplaceAllString(cleaned, "...")
	return cleaned
}

// ParseDate parses various date formats
func ParseDate(dateStr string) time.Time {
	layouts := []string{
		time.RFC1123Z, time.RFC1123, time.RFC3339, time.RFC822Z, time.RFC822,
		"Mon, 02 Jan 2006 15:04:05 -0700", "Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST", "Mon, 2 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z", "2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000Z", "2006-01-02 15:04:05",
		"Jan 2, 2006 3:04:05 PM", "January 2, 2006 3:04:05 PM",
		"January 2, 2006", "Jan 2, 2006", "2006-01-02",
		"02/01/2006 15:04:05", "02-01-2006 15:04:05",
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
		return fmt.Sprintf("%d minute(s) ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hour(s) ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d day(s) ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / (24 * 7))
		return fmt.Sprintf("%d week(s) ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / (24 * 30))
		return fmt.Sprintf("%d month(s) ago", months)
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
	readTime := wordCount / 225
	if readTime < 1 {
		return 1
	}
	return readTime
}

// CalculateScore calculates news item score for ranking
func CalculateScore(title, description string, stockSymbols []string, sourcePriority int) float64 {
	score := 0.0
	score += float64(5 - sourcePriority)
	score += float64(len(stockSymbols)) * 0.5

	titleLen := len(title)
	if titleLen >= 40 && titleLen <= 100 {
		score += 1.0
	} else if titleLen >= 20 && titleLen <= 150 {
		score += 0.5
	}

	descLen := len(description)
	if descLen >= 100 && descLen <= 500 {
		score += 1.0
	} else if descLen >= 50 {
		score += 0.5
	}

	sentiment := AnalyzeSentiment(title + " " + description)
	switch sentiment {
	case "positive", "negative":
		score += 0.3
	case "neutral":
		score += 0.1
	}

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

	categories := map[string][]string{
		"banking":     {"BANK", "SBI", "HDFC", "ICICI", "AXIS", "KOTAK"},
		"technology":  {"IT", "TECH", "TCS", "INFOSYS", "WIPRO", "SOFTWARE"},
		"energy":      {"OIL", "GAS", "ENERGY", "RELIANCE", "ONGC", "BPCL"},
		"automotive":  {"AUTO", "CAR", "MARUTI", "TATA MOTORS", "BAJAJ"},
		"pharma":      {"PHARMA", "DRUG", "MEDICINE", "SUN PHARMA", "DR REDDY"},
		"fmcg":        {"FMCG", "CONSUMER", "UNILEVER", "ITC", "NESTLE"},
		"metals":      {"STEEL", "METAL", "TATA STEEL", "JSW", "HINDALCO"},
		"telecom":     {"TELECOM", "MOBILE", "BHARTI", "AIRTEL", "JIO"},
		"realty":      {"REAL ESTATE", "PROPERTY", "REALTY", "DLF"},
		"mutual_fund": {"MUTUAL FUND", "MF", "SIP", "NAV"},
		"ipo":         {"IPO", "LISTING", "PUBLIC OFFERING"},
		"results":     {"RESULTS", "EARNINGS", "QUARTERLY", "ANNUAL"},
		"dividend":    {"DIVIDEND", "PAYOUT", "YIELD"},
		"merger":      {"MERGER", "ACQUISITION", "DEAL", "TAKEOVER"},
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

	sentiment := AnalyzeSentiment(text)
	if sentiment != "neutral" && !tagMap[sentiment] {
		tags = append(tags, sentiment)
		tagMap[sentiment] = true
	}

	if len(tags) > 5 {
		tags = tags[:5]
	}
	return tags
}

// FilterNews filters news based on given criteria
func FilterNews(news []NewsItem, filter FilterOptions) ([]NewsItem, int) {
	var filtered []NewsItem

	for _, item := range news {
		if filter.Source != "" && item.Source != filter.Source {
			continue
		}
		if filter.Category != "" && item.Category != filter.Category {
			continue
		}
		if filter.Sentiment != "" && item.Sentiment != filter.Sentiment {
			continue
		}
		if filter.StockOnly && !item.IsStockNews {
			continue
		}
		if !filter.DateFrom.IsZero() && item.PubDate.Before(filter.DateFrom) {
			continue
		}
		if !filter.DateTo.IsZero() && item.PubDate.After(filter.DateTo) {
			continue
		}
		if filter.MinScore > 0 && item.Score < filter.MinScore {
			continue
		}
		if len(filter.Keywords) > 0 {
			found := false
			for _, keyword := range filter.Keywords {
				if strings.Contains(strings.ToLower(item.Title), strings.ToLower(keyword)) ||
					strings.Contains(strings.ToLower(item.Description), strings.ToLower(keyword)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	total := len(filtered)

	// Sort if needed
	if filter.SortBy != "" {
		sort.Slice(filtered, func(i, j int) bool {
			if filter.SortOrder == "desc" {
				return filtered[i].Score > filtered[j].Score
			}
			return filtered[i].Score < filtered[j].Score
		})
	}

	// Apply pagination
	start := filter.Offset
	end := start + filter.Limit
	if start > total {
		return []NewsItem{}, total
	}
	if end > total {
		end = total
	}
	return filtered[start:end], total
}
