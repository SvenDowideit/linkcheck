# Web site link checker.

This tool will traverse the url you specify, parse the html, and then check that each URL linked on that
page is GET-able. Any linked pages that start with the same url fragment as your original request will
also be parsed.

So the following example will test all links on all pages that it finds on that site:

```
$ ./linkcheck http://some.url.com/
```

Whereas the next example will only test pages with url's that begin with `http://some.url.com/some/sub/section/`

```
$ ./linkcheck http://some.url.com/some/sub/section/
```


