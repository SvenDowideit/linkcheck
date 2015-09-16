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
	url  string
	code int
	err  error
}

type NewUrl struct {
	from string
	url  string
}

type FoundUrls struct {
	response   int
	usageCount int
	err        error
	// list of pages that refer to the link?
}

var foundUrls = make(map[string]FoundUrls)

func crawl(chWork chan NewUrl, ch chan NewUrl, chFinished chan UrlResponse) {
	for true {
		new := <-chWork
		crawlOne(new, ch, chFinished)
	}
}

// Extract all http** links from a given webpage
func crawlOne(req NewUrl, ch chan NewUrl, chFinished chan UrlResponse) {
	base, err := url.Parse(req.url)
	reply := UrlResponse{
		url: req.url,
		//from: req.from
		code: 999,
	}
	if err != nil {
		fmt.Println("ERROR: failed to Parse \"" + req.url + "\"")
		reply.err = err
		chFinished <- reply
		return
	}
	switch base.Scheme {
	case "mailto", "irc":
		reply.err = fmt.Errorf("%s on page %s", base.Scheme, req.from)
		reply.code = 900
		chFinished <- reply
		return
	}
	resp, err := http.Get(req.url)
	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + req.url + "\"  " + err.Error())
		reply.err = err
		chFinished <- reply
		return
	}
	defer func() {
		// Notify that we're done after this function
		reply.code = resp.StatusCode
		reply.err = fmt.Errorf("from %s", req.from)
		chFinished <- reply
	}()

	//fmt.Println("\t crawled \"" + req.url + "\"")

	b := resp.Body
	defer b.Close() // close Body when the function returns

	// only parse if this page is on the original site
	if !strings.HasPrefix(req.url, seedUrl) {
		return
	}

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
				from: req.url,
				url:  base.ResolveReference(u).String(),
			}
			ch <- new
		}
	}
}

var seedUrl = os.Args[1]

func main() {
	seedUrls := os.Args[1:]

	// Channels
	chUrls := make(chan NewUrl, 10000)
	chWork := make(chan NewUrl, 10000)
	chFinished := make(chan UrlResponse)

	for w := 1; w <= 10; w++ {
		go crawl(chWork, chUrls, chFinished)
	}

	for _, url := range seedUrls {
		new := NewUrl{
			from: "",
			url:  url,
		}
		chUrls <- new
	}

	// Subscribe to both channels
	count := 0
	for len(chUrls) > 0 || len(chWork) > 0 {
		select {
		case url := <-chUrls:
			// TODO: contemplate cacheing the html, or parse result to enable checking for anchor existance
			if f, ok := foundUrls[url.url]; !ok {
				count++
				foundUrls[url.url] = FoundUrls{
					usageCount: 1,
					response:   0,
				}
				chWork <- url
			} else {
				f.usageCount++
				foundUrls[url.url] = f
			}
			fmt.Printf("(w%d, w%d)", len(chWork), len(chUrls))

		case ret := <-chFinished:
			info := foundUrls[ret.url]
			info.response = ret.code
			info.err = ret.err
			foundUrls[ret.url] = info
		}
	}

	// We're done! Print the results...
	summary := make(map[int]int)
	for url, info := range foundUrls {
		summary[info.response]++
		if info.response != 200 && info.response != 900 {
			fmt.Printf(" - %d (%d): %s\n", info.response, info.usageCount, url)
			fmt.Printf("\t%s\n", info.err)
		}
	}
	fmt.Println("\nFound", len(foundUrls), "unique urls\n")
	for code, count := range summary {
		fmt.Printf("\t\tStatus %d : %d\n", code, count)
	}

	close(chUrls)
}
