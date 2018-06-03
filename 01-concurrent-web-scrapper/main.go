package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/html"
)

func main() {
	var (
		startURL  string
		rate      time.Duration
		timeLimit time.Duration
	)

	flag.StringVar(&startURL, "url", "", "URL to scrap")
	flag.DurationVar(&rate, "rate", 100*time.Millisecond, "time between requests")
	flag.DurationVar(&timeLimit, "time-limit", 10*time.Second, "time limit")

	flag.Parse()

	if startURL == "" {
		flag.Usage()
		fmt.Println("url is required")
		os.Exit(1)
	}

	s := &Scraper{}

	URL, err := url.Parse(startURL)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeLimit)
	defer cancel()
	URLs := s.Run(ctx, URL, rate)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("main: Context is done, returning...")
			return
		case URL = <-URLs:
			fmt.Println(URL.String())
		}
	}

}

type Scraper struct {
	visitedURLs    map[string]bool
	pendingURLs    chan *url.URL
	discoveredURLs chan *url.URL
}

func (s *Scraper) Run(ctx context.Context, URL *url.URL, rate time.Duration) chan *url.URL {
	s.visitedURLs = make(map[string]bool)
	s.pendingURLs = make(chan *url.URL, 10)
	s.discoveredURLs = make(chan *url.URL)
	throttle := time.Tick(rate)

	s.pendingURLs <- URL

	go func() {
		for pending := range s.pendingURLs {

			select {
			case <-ctx.Done():
				fmt.Println("scraper: Context is done, returning...")
				return
			default:
				if _, visited := s.visitedURLs[pending.String()]; !visited {
					s.visitedURLs[pending.String()] = true
					s.discoveredURLs <- pending
					<-throttle
					go s.visit(pending)
				}
			}
		}
	}()

	return s.discoveredURLs
}

func (s *Scraper) visit(URL *url.URL) {
	resp, err := http.Get(URL.String())

	if err != nil {
		log.Println(err)
		return
	}

	defer resp.Body.Close()
	tokenizer := html.NewTokenizer(resp.Body)

tokens:
	for tt := tokenizer.Next(); tt != html.ErrorToken; tt = tokenizer.Next() {

		if tt == html.StartTagToken {
			t := tokenizer.Token()

			if t.Data == "a" {
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						linkURL, err := url.Parse(attr.Val)

						if err != nil {
							log.Println(err)
							continue tokens
						}

						if !linkURL.IsAbs() {
							linkURL.Scheme = URL.Scheme
							linkURL.Host = URL.Host
						}

						if linkURL.Scheme == "http" || linkURL.Scheme == "https" {
							s.pendingURLs <- linkURL
						}
						continue tokens
					}
				}
			}
		}
	}
}
