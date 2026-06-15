package cmo

import (
	"bytes"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

var yearRE = regexp.MustCompile(`(\d{4})`)

// extractPDFLinks returns all PDF href values found in body using the
// golang.org/x/net/html tokenizer. Only href attributes on <a> tags that
// end with ".pdf" (case-insensitive) are returned.
func extractPDFLinks(body []byte) []string {
	var links []string
	z := html.NewTokenizer(bytes.NewReader(body))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		name, hasAttr := z.TagName()
		if !hasAttr || string(name) != "a" {
			continue
		}
		for {
			key, val, more := z.TagAttr()
			if string(key) == "href" {
				s := string(val)
				if strings.HasSuffix(strings.ToLower(s), ".pdf") {
					links = append(links, s)
				}
			}
			if !more {
				break
			}
		}
	}
	return links
}

// isCJMO reports whether the filename belongs to the Canadian Junior
// Mathematical Olympiad rather than the senior CMO.
func isCJMO(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.Contains(lower, "cjmo") ||
		strings.Contains(lower, "jexam") ||
		strings.Contains(lower, "jsol")
}

// isSolutions reports whether the filename is a solutions PDF rather than a
// problems PDF.
func isSolutions(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.Contains(lower, "sol")
}

// classifyURL extracts year, competition ("CMO"/"CJMO"), and type
// ("problems"/"solutions") from a PDF URL.
// Returns year=0 when no 4-digit year can be found in the filename.
func classifyURL(rawURL string) (year int, competition, docType string) {
	base := path.Base(rawURL)
	// strip extension
	name := strings.TrimSuffix(base, path.Ext(base))

	// find first 4-digit number in the filename (not the upload-path year)
	m := yearRE.FindString(name)
	if m == "" {
		return 0, "", ""
	}
	y, _ := strconv.Atoi(m)
	if y < 1969 || y > 2030 {
		return 0, "", ""
	}

	if isCJMO(base) {
		competition = "CJMO"
	} else {
		competition = "CMO"
	}
	if isSolutions(base) {
		docType = "solutions"
	} else {
		docType = "problems"
	}
	return y, competition, docType
}

// buildEditions groups classified PDF URLs into Edition records sorted
// newest-first. Editions with no discoverable year are silently skipped.
func buildEditions(urls []string) []Edition {
	type key struct {
		year        int
		competition string
	}
	groups := map[key]*Edition{}
	order := []key{}

	for _, u := range urls {
		year, competition, docType := classifyURL(u)
		if year == 0 {
			continue
		}
		k := key{year, competition}
		if _, ok := groups[k]; !ok {
			groups[k] = &Edition{Year: year, Competition: competition}
			order = append(order, k)
		}
		ed := groups[k]
		switch docType {
		case "problems":
			if ed.ProblemsURL == "" {
				ed.ProblemsURL = u
			}
		case "solutions":
			if ed.SolutionsURL == "" {
				ed.SolutionsURL = u
			}
		}
	}

	// Sort descending by year, then CMO before CJMO within same year.
	// "CMO" > "CJMO" lexicographically (M > J), so descending comp sort
	// puts CMO first.
	sort.Slice(order, func(i, j int) bool {
		ki, kj := order[i], order[j]
		if ki.year != kj.year {
			return ki.year > kj.year
		}
		return ki.competition > kj.competition // "CMO" > "CJMO"
	})

	out := make([]Edition, 0, len(order))
	for rank, k := range order {
		ed := *groups[k]
		ed.Rank = rank + 1
		out = append(out, ed)
	}
	return out
}
