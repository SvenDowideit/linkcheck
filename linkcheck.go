package main

import (
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Helper function to pull the href attribute from a Token
func getHref(t html.Token) (ok bool, href string) {
	// Iterate over all of the Token's attributes until we find an "href"
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}

	// "bare" return will return the variables (ok, href) as defined in
	// the function definition
	return
}

type UrlResponse struct {
	url string
	code int
}

type NewUrl struct {
	from string
	url string
}


// Extract all http** links from a given webpage
func crawl(reqUrl string, ch chan NewUrl, chFinished chan UrlResponse) {
	base, err := url.Parse(reqUrl)
	reply := UrlResponse {
		url: reqUrl,
		code: 999,
	}
	if err != nil {
		fmt.Println("ERROR: failed to Parse \"" + reqUrl + "\"")
		chFinished <- reply
		return
	}
	resp, err := http.Get(reqUrl)
	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + reqUrl + "\"  " + err.Error())
		chFinished <- reply
		return
	}
	defer func() {
		// Notify that we're done after this function
		reply.code = resp.StatusCode
		chFinished <- reply
	}()


	fmt.Println("\t crawled \"" + reqUrl + "\"")

	b := resp.Body
	defer b.Close() // close Body when the function returns

	z := html.NewTokenizer(b)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()

			// Check if the token is an <a> tag
			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}

			// Extract the href value, if there is one
			ok, newUrl := getHref(t)
			if !ok {
				continue
			}
			u, e := url.Parse(newUrl)
			if e != nil {
				fmt.Println("ERROR: failed to Parse \"" + newUrl + "\"")
				continue
			}
			new := NewUrl{
				from: reqUrl,
				url: base.ResolveReference(u).String(),
				}
			ch <- new
		}
	}
}

func main() {
	foundUrls := make(map[string]int)
	seedUrls := os.Args[1:]

	// Channels
	chUrls := make(chan NewUrl)
	chFinished := make(chan UrlResponse)

	// Kick off the crawl process (concurrently)
	for _, url := range seedUrls {
		foundUrls[url] = 0
		go crawl(url, chUrls, chFinished)
	}

	// Subscribe to both channels
	count := len(seedUrls)
	for c := 0; c < count; {
		select {
		case new := <-chUrls:
			if _, ok := foundUrls[new.url] ; !ok {
				// TODO: need to stick to only the site we're testing plus one
				if strings.HasPrefix(new.from, seedUrls[0]) {
				count++
				// TODO: you're kidding right - lets not make an infinite number of cawlers?
				go crawl(new.url, chUrls, chFinished)
				foundUrls[new.url] = 0
				}
			}
		case ret := <-chFinished:
			foundUrls[ret.url] = ret.code
			c++
		}
	}

	// We're done! Print the results...
	for url, code := range foundUrls {
		fmt.Printf(" - %d: %s\n", code, url)
	}
	fmt.Println("\nFound", len(foundUrls), "unique urls:\n")

	close(chUrls)
}
