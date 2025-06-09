package ratelimit

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu        sync.Mutex
	capacity int           // Maximum number of requests allowed in the time window
	interval time.Duration // Time window for rate limiting
	tokens   int           // Current number of available tokens
	lastTime time.Time     // Last time tokens were updated
}

// NewRateLimiter creates a new RateLimiter that allows up to requestsPerMinute requests per minute
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60 // Default to 1 request per second
	}
	return &RateLimiter{
		capacity: requestsPerMinute,
		interval: time.Minute,
		tokens:   requestsPerMinute,
		lastTime: time.Now(),
	}
}

// Wait blocks until the request is allowed to proceed
func (rl *RateLimiter) Wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Calculate how many tokens to add based on time elapsed
	elapsed := now.Sub(rl.lastTime)
	if elapsed > rl.interval {
		// More than interval has passed, reset tokens to full capacity
		rl.tokens = rl.capacity
		rl.lastTime = now
	} else {
		// Calculate how many tokens to add based on elapsed time
		tokensToAdd := int(float64(rl.capacity) * (float64(elapsed) / float64(rl.interval)))
		if tokensToAdd > 0 {
			// Add tokens but don't exceed capacity
			rl.tokens = min(rl.tokens+tokensToAdd, rl.capacity)
			rl.lastTime = now
		}
	}

	// If no tokens available, wait until the next token is available
	if rl.tokens <= 0 {
		// Calculate when the next token will be available
		timeToNextToken := rl.lastTime.Add(time.Duration(float64(rl.interval) / float64(rl.capacity))).Sub(now)
		time.Sleep(timeToNextToken)
		
		// After waiting, update the state
		now = time.Now()
		rl.lastTime = now
		rl.tokens = rl.capacity - 1 // Use one token for this request
		return
	}

	// Use a token for this request
	rl.tokens--
	rl.lastTime = now
}

// SetRate updates the rate limiter's maximum requests per minute
func (rl *RateLimiter) SetRate(requestsPerMinute int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if requestsPerMinute <= 0 {
		requestsPerMinute = 60 // Default to 1 request per second
	}

	ratio := float64(requestsPerMinute) / float64(rl.capacity)
	rl.capacity = requestsPerMinute
	rl.tokens = int(float64(rl.tokens) * ratio)
}

// GetRate returns the current maximum requests per minute
func (rl *RateLimiter) GetRate() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return rl.capacity
}

// GetTokens returns the current number of available tokens
func (rl *RateLimiter) GetTokens() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return rl.tokens
}

// Reset resets the rate limiter to its initial state
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.tokens = rl.capacity
	rl.lastTime = time.Now()
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
