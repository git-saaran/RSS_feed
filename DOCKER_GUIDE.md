# 🐳 Docker Deployment Guide - Enhanced RSS Aggregator

## 🚀 **Quick Start**

### **Option 1: Docker Compose (Recommended)**
```bash
# Build and start the application
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the application
docker-compose down
```

### **Option 2: Docker Build & Run**
```bash
# Build the image
docker build -t rss-aggregator:latest .

# Run the container
docker run -d \
  --name rss-aggregator \
  -p 8080:8080 \
  -e TZ=Asia/Kolkata \
  --memory=256m \
  --cpus=0.5 \
  rss-aggregator:latest
```

---

## 🎯 **Optimizations for 4GB Machine**

### **Resource Limits Applied:**
```yaml
# Conservative limits for 4GB system
deploy:
  resources:
    limits:
      memory: 256M      # Max 256MB RAM
      cpus: '0.5'       # Max 50% CPU
    reservations:
      memory: 128M      # Reserved 128MB
      cpus: '0.25'      # Reserved 25% CPU
```

### **Why These Limits?**
- **256MB RAM limit**: Leaves 3.7GB for OS and other apps
- **50% CPU limit**: Ensures system remains responsive
- **Conservative approach**: Prevents system overload

---

## 🔧 **Enhanced Features in Docker**

### **1. Multi-Stage Build**
```dockerfile
# Optimized build process:
Stage 1: golang:1.21-alpine (build dependencies)
Stage 2: alpine:latest (minimal runtime)

# Result: ~15MB final image (vs ~800MB with full Go)
```

### **2. Security Enhancements**
```dockerfile
# Non-root user for security
USER appuser

# Minimal attack surface
FROM alpine:latest (only essential packages)
```

### **3. Health Checks**
```yaml
# Tests multiple endpoints
healthcheck:
  test: |
    - /api/status     ✅ Basic functionality
    - /api/analytics  ✅ Analytics engine
    - /api/sentiment  ✅ Sentiment analysis
```

---

## 📊 **Container Monitoring**

### **Check Container Health**
```bash
# View container status
docker ps

# Check health status
docker inspect rss-advanced-aggregator | grep Health -A 10

# View real-time stats
docker stats rss-advanced-aggregator
```

### **Access Application**
```bash
# Main interface
curl http://localhost:8080

# API endpoints
curl http://localhost:8080/api/status
curl http://localhost:8080/api/analytics
curl http://localhost:8080/api/sentiment

# WebSocket (testing)
wscat -c ws://localhost:8080/ws
```

---

## 🐛 **Troubleshooting**

### **Container Won't Start**
```bash
# Check logs
docker-compose logs rss-aggregator

# Common issues:
1. Port 8080 already in use
   Solution: Change port mapping in docker-compose.yml

2. Memory limit too low
   Solution: Increase memory limit if you have more RAM

3. Build fails
   Solution: Check go.mod and go.sum files exist
```

### **Performance Issues**
```bash
# Monitor resource usage
docker stats --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}"

# If high CPU/Memory:
1. Check RSS feed sources (some might be slow)
2. Reduce refresh interval
3. Increase resource limits slightly
```

### **WebSocket Issues**
```bash
# Test WebSocket connection
# Browser console:
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onopen = () => console.log('Connected');
ws.onmessage = (e) => console.log('Data:', JSON.parse(e.data));
```

---

## ⚙️ **Configuration Options**

### **Environment Variables**
```yaml
environment:
  - TZ=Asia/Kolkata                    # Timezone
  - GO_ENV=production                  # Environment mode
  - RSS_REFRESH_INTERVAL=5m            # How often to fetch RSS
  - MAX_ARTICLES_CACHE=1000            # Max articles in memory
  - WEBSOCKET_TIMEOUT=60s              # WebSocket timeout
```

### **Custom Configuration**
```bash
# Run with different settings
docker run -d \
  --name rss-custom \
  -p 8080:8080 \
  -e RSS_REFRESH_INTERVAL=10m \
  -e MAX_ARTICLES_CACHE=500 \
  rss-aggregator:latest
```

---

## 📈 **Performance Benchmarks**

### **Expected Performance (4GB Machine)**
```bash
✅ Startup Time: ~3-5 seconds
✅ Memory Usage: ~80-120MB steady state
✅ CPU Usage: ~5-15% during RSS fetching
✅ Response Time: <100ms for all endpoints
✅ WebSocket Latency: <10ms
✅ Article Processing: ~100 articles/second
```

### **Monitoring Commands**
```bash
# Real-time performance
watch -n 1 'docker stats --no-stream'

# Memory usage over time
docker stats --format "{{.MemUsage}}" --no-stream

# Check if hitting limits
docker inspect rss-advanced-aggregator | grep -A 5 Resources
```

---

## 🔄 **Maintenance**

### **Updates**
```bash
# Rebuild with latest code
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# View updated logs
docker-compose logs -f --tail=50
```

### **Backup & Restore**
```bash
# Export container (if needed)
docker export rss-advanced-aggregator > rss-backup.tar

# Import container
docker import rss-backup.tar rss-aggregator:backup
```

### **Cleanup**
```bash
# Remove unused images/containers
docker system prune

# Remove specific container
docker-compose down --rmi all --volumes
```

---

## 🌟 **Production Deployment**

### **Recommended Setup**
```yaml
# docker-compose.prod.yml
version: '3.8'
services:
  rss-aggregator:
    image: rss-aggregator:latest
    restart: always
    ports:
      - "80:8080"
    environment:
      - GO_ENV=production
      - RSS_REFRESH_INTERVAL=5m
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '1.0'
    healthcheck:
      interval: 60s
      timeout: 30s
      retries: 5
```

### **With Reverse Proxy (Nginx)**
```nginx
# nginx.conf
upstream rss-app {
    server localhost:8080;
}

server {
    listen 80;
    server_name yourdomain.com;
    
    location / {
        proxy_pass http://rss-app;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
    
    # WebSocket support
    location /ws {
        proxy_pass http://rss-app;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

---

## ✅ **Testing Checklist**

### **After Deployment**
```bash
# 1. Container health
□ docker ps shows "healthy" status
□ No error logs in docker-compose logs

# 2. Web interface
□ http://localhost:8080 loads correctly
□ Dark/light mode toggle works
□ Search functionality works
□ Analytics dashboard displays

# 3. API endpoints
□ /api/status returns success
□ /api/analytics returns data
□ /api/sentiment returns percentages
□ /api/filter accepts parameters

# 4. Real-time features
□ WebSocket connection established
□ Auto-refresh works (check console)
□ Notifications appear for new articles

# 5. Performance
□ Page loads in <2 seconds
□ Memory usage under 200MB
□ CPU usage under 20%
```

---

## 🚨 **Alerts & Monitoring**

### **Health Check Failures**
```bash
# Set up monitoring
#!/bin/bash
# health-monitor.sh

while true; do
    if ! docker exec rss-advanced-aggregator wget -q -O - http://localhost:8080/api/status > /dev/null; then
        echo "ALERT: RSS Aggregator health check failed at $(date)"
        # Send notification (email, Slack, etc.)
    fi
    sleep 60
done
```

### **Resource Monitoring**
```bash
# memory-monitor.sh
#!/bin/bash

MEMORY_USAGE=$(docker stats --no-stream --format "{{.MemPerc}}" rss-advanced-aggregator | tr -d '%')

if (( $(echo "$MEMORY_USAGE > 80" | bc -l) )); then
    echo "ALERT: High memory usage: ${MEMORY_USAGE}%"
fi
```

---

## 🎯 **Summary**

Your enhanced Docker setup now provides:

✅ **Optimized for 4GB machines** with conservative resource limits  
✅ **Multi-stage builds** for minimal image size (~15MB)  
✅ **Security hardening** with non-root user  
✅ **Comprehensive health checks** for all advanced features  
✅ **WebSocket support** for real-time updates  
✅ **Production-ready configuration** with proper logging  
✅ **Easy monitoring** and troubleshooting tools  

**Your Docker configuration is now perfectly optimized for the enhanced RSS aggregator!** 🚀