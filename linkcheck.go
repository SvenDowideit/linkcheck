package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/html"
)

func getAttr(t html.Token, name string) (ok bool, href string) {
	for _, a := range t.Attr {
		if a.Key == name {
			href = a.Val
			ok = true
		}
	}

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

var skipUrls = map[string]int{
	"https://build.opensuse.org/project/show/Virtualization:containers": 1,
	"http://10.0.0.2:5000":                                              1,
	"https://linux.oracle.com":                                          1,
	"http://supervisord.org/":                                           1,
	"http://goo.gl/HSz8UT":                                              1,
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
	if _, ok := skipUrls[req.url]; ok {
		fmt.Printf("Skipping: %s\n", req.url)
		reply.code = 299
		chFinished <- reply
		return
	}
	fmt.Printf("Crawling: %s\n", req.url)
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

	loc, err := resp.Location()
	if err == nil && req.url != loc.String() {
		fmt.Printf("\t crawled \"%s\"", req.url)
		fmt.Printf("\t\t to \"%s\"", loc)
	}

	b := resp.Body
	defer b.Close() // close Body when the function returns

	// only parse if this page is on the original site
	// if we moved this check back to the main loop, we'd parse more sites
	if !strings.HasPrefix(req.url, seedUrl) {
		return
	}

	// don't parse js files..
	if strings.HasSuffix(req.url, ".js") || strings.HasSuffix(req.url, ".js") {
		return
	}

	// TODO: it seems to sucessfully parse non-html (like js/css)
	z := html.NewTokenizer(b)

	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			// End of the document, we're done
			return
		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()
			var ok bool
			var newUrl string

			switch t.Data {
			case "base":
				// use the actual baseUrl set in the html file
				ok, baseUrl := getAttr(t, "href")
				if !ok {
					continue
				}

				newBase, err := url.Parse(baseUrl)
				if err != nil {
					continue
				}
				base = base.ResolveReference(newBase)
				continue
			case "a", "link":
				ok, newUrl = getAttr(t, "href")
				if !ok {
					continue
				}
			case "img", "script":
				ok, newUrl = getAttr(t, "src")
				if !ok {
					continue
				}
			default:
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

var seedUrl string

type FoundUrls struct {
	response   int
	usageCount int
	err        error
	from       map[string]int
}

func main() {
	if len(os.Args[1:]) == 0 {
		seedUrl = os.Getenv("CHECKURL")
		if seedUrl == "" {
			fmt.Println("Please specify a URL to check")
			os.Exit(-1)
		}
	} else {
		seedUrl = os.Args[1]
	}

	// Channels
	chUrls := make(chan NewUrl, 1000)
	chWork := make(chan NewUrl, 3000)
	chFinished := make(chan UrlResponse)

	var foundUrls = make(map[string]FoundUrls)

	for w := 1; w <= 20; w++ {
		go crawl(chWork, chUrls, chFinished)
	}

	new := NewUrl{
		from: "",
		url:  seedUrl,
	}
	chUrls <- new

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
	errorCount := 0
	explain := map[int]string{
		900: "mailto or irc",
		299: "url skipped",
		200: "ok",
		404: "forbidden",
		403: "forbidden",
	}
	for code, count := range summary {
		reason, ok := explain[code]
		if !ok {
			// TODO: I presume go has a text error mapping
			reason = "HTTP code"
		}
		fmt.Printf("\t\tStatus %d : %d - %s\n", code, count, reason)
		if code != 200 && code != 299 && code != 900 {
			errorCount += count
		}
	}
	fmt.Println("\nError Count:", errorCount)

	close(chUrls)

	// return the number of 404's to show that there are things to be fixed
	os.Exit(errorCount)
}
