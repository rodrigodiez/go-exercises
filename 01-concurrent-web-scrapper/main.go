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
		baseURL        string
		concurrentCons int
		rate           time.Duration
		URL            *url.URL
		foundURLS      chan *url.URL
		visitedURLS    map[string]bool
		s              *scraper
		timeLimit      time.Duration
	)

	flag.StringVar(&baseURL, "url", "", "URL to scrap")
	flag.IntVar(&concurrentCons, "concurrent", 10, "concurrent requests")
	flag.DurationVar(&rate, "rate", 100*time.Millisecond, "time between requests")
	flag.DurationVar(&timeLimit, "time-limit", 10*time.Second, "time limit")

	flag.Parse()

	if baseURL == "" {
		flag.Usage()
		fmt.Println("url is required")
		os.Exit(1)
	}

	foundURLS = make(chan *url.URL, concurrentCons)
	visitedURLS = make(map[string]bool)
	s = &scraper{urls: foundURLS}

	URL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatal(err)
	}

	go s.Run(URL)

	throttle := time.Tick(rate)
	ctx, cancel := context.WithTimeout(context.Background(), timeLimit)
	defer cancel()

	for URL = range foundURLS {
		select {
		case <-ctx.Done():
			fmt.Printf("Found %d URLs\n", len(visitedURLS))
			return
		default:
		}

		if _, exists := visitedURLS[URL.String()]; !exists {
			<-throttle
			visitedURLS[URL.String()] = true
			fmt.Println(URL.String())
			go s.Run(URL)
		}

	}
}

type scraper struct {
	urls chan *url.URL
}

func (s *scraper) Run(URL *url.URL) {
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
							s.urls <- linkURL
						}
						continue tokens
					}
				}
			}
		}
	}
}
