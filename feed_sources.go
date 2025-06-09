package main

import (
	"time"
)

// Default feed sources configuration
func GetDefaultFeedSources() map[string]FeedSource {
	return map[string]FeedSource{
		"economic-times": {
			ID:         "economic-times",
			Name:       "Economic Times",
			URL:        "https://economictimes.indiatimes.com/rssfeedstopstories.cms",
			Logo:       "ET",
			Class:      "economic-times",
			Enabled:    true,
			Priority:   1,
			Category:   "business",
			Language:   "en",
			Country:    "IN",
			UpdateFreq: 15,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		"hindu-business": {
			ID:         "hindu-business",
			Name:       "The Hindu Business",
			URL:        "https://www.thehindu.com/business/feeder/default.rss",
			Logo:       "TH",
			Class:      "hindu",
			Enabled:    true,
			Priority:   2,
			Category:   "business",
			Language:   "en",
			Country:    "IN",
			UpdateFreq: 20,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		"financial-express": {
			ID:         "financial-express",
			Name:       "Financial Express",
			URL:        "https://www.financialexpress.com/rss/",
			Logo:       "FE",
			Class:      "financial-express",
			Enabled:    true,
			Priority:   3,
			Category:   "finance",
			Language:   "en",
			Country:    "IN",
			UpdateFreq: 10,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		"moneycontrol": {
			ID:         "moneycontrol",
			Name:       "MoneyControl",
			URL:        "https://www.moneycontrol.com/rss/business.xml",
			Logo:       "MC",
			Class:      "moneycontrol",
			Enabled:    true,
			Priority:   2,
			Category:   "finance",
			Language:   "en",
			Country:    "IN",
			UpdateFreq: 15,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		"business-standard": {
			ID:         "business-standard",
			Name:       "Business Standard",
			URL:        "https://www.business-standard.com/rss/finance-101.rss",
			Logo:       "BS",
			Class:      "business-standard",
			Enabled:    true,
			Priority:   3,
			Category:   "finance",
			Language:   "en",
			Country:    "IN",
			UpdateFreq: 20,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		"livemint": {
			ID:         "livemint",
			Name:       "LiveMint",
			URL:        "https://www.livemint.com/rss/money",
			Logo:       "LM",
			Class:      "livemint",
			Enabled:    true,
			Priority:   2,
			Category:   "finance",
			Language:   "en",
			Country:    "IN",
			UpdateFreq: 10,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		"reuters-business": {
			ID:         "reuters-business",
			Name:       "Reuters Business",
			URL:        "https://feeds.reuters.com/reuters/businessNews",
			Logo:       "RT",
			Class:      "reuters",
			Enabled:    true,
			Priority:   1,
			Category:   "international",
			Language:   "en",
			Country:    "US",
			UpdateFreq: 5,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		"bloomberg": {
			ID:         "bloomberg",
			Name:       "Bloomberg",
			URL:        "https://feeds.bloomberg.com/markets/news.rss",
			Logo:       "BB",
			Class:      "bloomberg",
			Enabled:    false, // May require special handling
			Priority:   1,
			Category:   "markets",
			Language:   "en",
			Country:    "US",
			UpdateFreq: 5,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		"cnbc": {
			ID:         "cnbc",
			Name:       "CNBC",
			URL:        "https://www.cnbc.com/id/100003114/device/rss/rss.html",
			Logo:       "CNBC",
			Class:      "cnbc",
			Enabled:    true,
			Priority:   2,
			Category:   "markets",
			Language:   "en",
			Country:    "US",
			UpdateFreq: 10,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		"marketwatch": {
			ID:         "marketwatch",
			Name:       "MarketWatch",
			URL:        "https://feeds.marketwatch.com/marketwatch/topstories/",
			Logo:       "MW",
			Class:      "marketwatch",
			Enabled:    true,
			Priority:   3,
			Category:   "markets",
			Language:   "en",
			Country:    "US",
			UpdateFreq: 15,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}
}

// Stock keywords for Indian and international markets
var StockKeywords = map[string][]string{
	"indian_stocks": {
		// Banking
		"SBI", "HDFCBANK", "ICICIBANK", "AXISBANK", "KOTAKBANK", "INDUSINDBK",
		"BANKBARODA", "PNB", "CANBK", "IDFCFIRSTB",

		// IT
		"TCS", "INFOSYS", "WIPRO", "HCLTECH", "TECHM", "MINDTREE", "MPHASIS",
		"LTI", "COFORGE", "PERSISTENT",

		// Energy & Oil
		"RELIANCE", "ONGC", "BPCL", "HPCL", "IOC", "GAIL", "NTPC", "POWERGRID",
		"COALINDIA", "ADANIGREEN", "TATAPOWER",

		// Auto
		"MARUTI", "TATAMOTORS", "M&M", "BAJAJ-AUTO", "HEROMOTOCO", "EICHERMOT",
		"ASHOKLEY", "TVSMOTOR", "BAJAJFINSV",

		// FMCG
		"HINDUNILVR", "ITC", "NESTLEIND", "BRITANNIA", "DABUR", "MARICO",
		"COLPAL", "GODREJCP", "EMAMILTD",

		// Pharma
		"SUNPHARMA", "DRREDDY", "CIPLA", "DIVISLAB", "BIOCON", "CADILAHC",
		"AUROPHARMA", "LUPIN", "TORNTPHARM",

		// Metals & Mining
		"TATASTEEL", "JSWSTEEL", "HINDALCO", "VEDL", "SAIL", "NMDC",
		"COALINDIA", "HINDZINC", "RATNAMANI",

		// Cement
		"ULTRACEMCO", "SHREECEM", "ACC", "AMBUJACEMENT", "HEIDELBERG",
		"JKCEMENT", "RAMCOCEM",

		// Others
		"LT", "ASIANPAINT", "TITAN", "BHARTIARTL", "ADANIPORTS", "ADANIENT",
		"GRASIM", "UPL", "BAJFINANCE", "HDFCLIFE", "SBILIFE", "ICICIPRULI",
	},

	"international_stocks": {
		// US Tech Giants
		"AAPL", "MSFT", "GOOGL", "GOOG", "AMZN", "TSLA", "META", "NVDA",
		"NFLX", "CRM", "ORCL", "INTC", "AMD", "QCOM", "AVGO",

		// US Banks
		"JPM", "BAC", "WFC", "GS", "MS", "C", "USB", "PNC", "TFC", "COF",

		// Other Major US
		"BRK.A", "BRK.B", "JNJ", "V", "MA", "UNH", "HD", "PG", "DIS", "ADBE",
		"VZ", "T", "XOM", "CVX", "PFE", "KO", "PEP", "WMT", "COST",
	},

	"indices": {
		"NIFTY", "SENSEX", "BSE", "NSE", "NIFTY50", "BANKNIFTY", "NIFTYNEXT50",
		"DOW", "NASDAQ", "S&P500", "FTSE", "DAX", "NIKKEI", "HANG SENG",
	},

	"crypto": {
		"BITCOIN", "BTC", "ETHEREUM", "ETH", "DOGECOIN", "DOGE", "ADA", "DOT",
		"LINK", "LTC", "BCH", "XRP", "BNB", "USDT", "USDC", "BUSD",
	},

	"general_finance": {
		"STOCK", "EQUITY", "SHARE", "MARKET", "TRADING", "IPO", "DIVIDEND",
		"BULL", "BEAR", "RALLY", "CRASH", "VOLATILITY", "EARNINGS", "REVENUE",
		"PROFIT", "LOSS", "QUARTER", "ANNUAL", "GROWTH", "DECLINE", "SURGE",
		"INVESTMENT", "PORTFOLIO", "MUTUAL FUND", "ETF", "BONDS", "COMMODITIES",
		"FOREX", "CURRENCY", "RUPEE", "DOLLAR", "EURO", "YEN", "POUND",
	},
}

// Sentiment keywords for news analysis
var SentimentKeywords = map[string][]string{
	"positive": {
		"surge", "soar", "jump", "leap", "climb", "rise", "gain", "profit",
		"growth", "boom", "bull", "rally", "upbeat", "optimistic", "strong",
		"robust", "healthy", "improve", "recover", "breakthrough", "success",
		"achievement", "milestone", "record", "high", "peak", "outperform",
		"beat", "exceed", "positive", "good", "excellent", "outstanding",
	},
	"negative": {
		"crash", "plunge", "dive", "tumble", "fall", "drop", "decline", "loss",
		"slump", "bear", "recession", "downturn", "weak", "poor", "disappointing",
		"miss", "underperform", "struggle", "challenge", "concern", "worry",
		"fear", "panic", "uncertainty", "volatility", "risk", "threat",
		"crisis", "problem", "issue", "trouble", "difficult", "hard",
	},
	"neutral": {
		"stable", "steady", "unchanged", "flat", "maintain", "hold", "continue",
		"expected", "forecast", "estimate", "predict", "analyst", "report",
		"announce", "declare", "state", "confirm", "update", "review",
	},
}
