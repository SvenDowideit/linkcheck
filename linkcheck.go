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
	from string
	url  string
	code int
	err  error
}

type NewUrl struct {
	from string
	url  string
}

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
		url:  req.url,
		from: req.from,
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
		chFinished <- reply
	}()

	//fmt.Println("\t crawled \"" + req.url + "\"")

	b := resp.Body
	defer b.Close() // close Body when the function returns

	// only parse if this page is on the original site
	// if we moved this check back to the main loop, we'd parse more sites
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

type FoundUrls struct {
	response   int
	usageCount int
	err        error
	from       map[string]int
}

func main() {
	seedUrls := os.Args[1:]

	// Channels
	chUrls := make(chan NewUrl, 1000)
	chWork := make(chan NewUrl, 1000)
	chFinished := make(chan UrlResponse)

	var foundUrls = make(map[string]FoundUrls)

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
	for len(chUrls) > 0 || count > 0 {
		select {
		case foundUrl := <-chUrls:
			// don't need to check err - its already been checked before its put in the chUrls que
			u, _ := url.Parse(foundUrl.url)
			// TODO: need a different pipeline for ensuring anchor fragments exist
			// TODO: consider only removing the query/fragment for docs urls
			u.RawQuery = ""
			u.Fragment = ""
			resourceUrl := u.String()

			f, ok := foundUrls[resourceUrl]
			if !ok {
				count++
				f.usageCount = 0
				f.response = 0
				f.from = make(map[string]int)
				f.from[foundUrl.from] = 1
				chWork <- NewUrl{
					from: foundUrl.from,
					url:  resourceUrl,
				}
			}
			f.usageCount++
			f.from[foundUrl.from]++
			foundUrls[resourceUrl] = f

		case ret := <-chFinished:
			count--
			info := foundUrls[ret.url]
			//info.from[ret.from]++
			info.response = ret.code
			info.err = ret.err
			foundUrls[ret.url] = info
		}
		// fmt.Printf("(w%d, u%d, c%d)", len(chWork), len(chUrls), count)
	}

	// We're done! Print the results...
	fmt.Println("\nDone.")
	summary := make(map[int]int)
	for url, info := range foundUrls {
		summary[info.response]++
		if info.response != 200 && info.response != 900 {
			fmt.Printf(" - %d (%d): %s\n", info.response, info.usageCount, url)
			if info.err != nil {
				fmt.Printf("\t%s\n", info.err)
			}
			for from, count := range info.from {
				fmt.Printf("\t\t%d times from %s\n", count, from)
			}
		}
	}
	fmt.Println("\nFound", len(foundUrls), "unique urls\n")
	for code, count := range summary {
		fmt.Printf("\t\tStatus %d : %d\n", code, count)
	}

	close(chUrls)

	// return the number of 404's to show that there are things to be fixed
	os.Exit(summary[404])
}
