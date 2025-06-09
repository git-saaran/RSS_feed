# RSS Feed Dashboard

A real-time RSS feed aggregator and dashboard built with Go. This application fetches news from multiple RSS feeds, analyzes the content, and presents it in a clean, web-based interface.

## Features

- Real-time RSS feed aggregation
- Web-based dashboard with a clean, responsive UI
- RESTful API for programmatic access
- Built-in rate limiting and caching
- Health monitoring and statistics
- Support for multiple feed sources
- Sentiment analysis of news content

## Prerequisites

- Go 1.16 or higher
- Git

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/RSS_feed.git
   cd RSS_feed
   ```

2. Build the application:
   ```bash
   go build -o rss-feed ./cmd/server
   ```

## Configuration

Create a `config.json` file in the project root with the following structure:

```json
{
  "port": ":8080",
  "poll_interval": "5m",
  "request_timeout": "30s",
  "server_timeout": "30s",
  "max_news_items": 1000,
  "enable_sentiment": true,
  "log_level": "info",
  "database_path": "./data/news.db",
  "cache_timeout": "10m",
  "max_concurrent": 10,
  "rate_limit_rpm": 60
}
```

## Running the Application

```bash
./rss-feed
```

The dashboard will be available at `http://localhost:8080`

## API Endpoints

- `GET /` - Web dashboard
- `GET /api/health` - Health check
- `GET /api/news` - Get news items
- `GET /api/feeds` - List feed sources
- `POST /api/feeds/refresh` - Refresh all feeds
- `GET /api/stats` - Get dashboard statistics

## Development

### Adding New Feed Sources

To add a new feed source, update the `GetDefaultFeedSources` function in `internal/feed/feed.go`.

### Running Tests

```bash
go test ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.