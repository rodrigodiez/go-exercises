#01-concurrent-web-scrapper

# Acceptance criteria
- Scraper finds all links in a given URL
- Scraper subsequentenly visits found links
- Scraper only visits each URL once
- Scraper honours a time limit
- Scraper honours a rate limit

# Run
```
go run main.go --url=https://news.ycombinator.com/ --rate=10ms --time-limit=5s