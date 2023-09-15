package main

import (
	"github.com/PuerkitoBio/goquery"
	"log"
	"regexp"
	"strings"
)

func pretreatment(bytes []byte) []byte {
	htmlRegex := regexp.MustCompile(`(?si)<\s*html\s*.*>.*<\s*/\s*html\s*>`)
	xStr := string(bytes)
	if htmlRegex.MatchString(xStr) {

		str := getVisibleText(xStr)
		return []byte(str)
	} else {
		return bytes
	}
}
func compressSpacesAndNewlines(input string) string {
	// 去除连续的空格和换行符
	spaceRegex := regexp.MustCompile(`\s+`)
	compressed := spaceRegex.ReplaceAllString(input, " ")

	// 去除行首和行尾的空格和换行符
	compressed = strings.TrimSpace(compressed)

	return compressed
}
func getVisibleText(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Fatal(err)
	}

	var visibleText strings.Builder
	doc.Find("body").Each(func(i int, s *goquery.Selection) {
		println(s)
		visibleText.WriteString(s.Text())
	})

	return visibleText.String()
}

//func main() {
//	str := "<html><body><h1>Hello, World!</h1></body></html>"
//
//	println(string(pretreatment([]byte(str))))
//
//}
