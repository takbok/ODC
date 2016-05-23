package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type docxLink struct {
	docxID      int
	docxUrl     string
	docxPageNum int
}

func main() {

	var in io.Reader

	switch len(os.Args[1:]) {
	case 1:
		file, err := os.Open(os.Args[1])
		if err != nil {
			fatal(err)
		}
		defer file.Close()
		in = file
	case 0:
		in = os.Stdin
	default:
		fatal(fmt.Errorf("wrong number of arguments"))
	}

	links := collect(in)

	rch := make(chan linkCheckResult)

	for _, link := range links {
		go check(link, rch)
	}

	linkCheckResults := make(map[int]linkCheckResult, len(links))

	for range links {
		r := <-rch
		linkCheckResults[r.docxLink.docxID] = r
	}
	outputReport(linkCheckResults)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

type linkCheckResult struct {
	docxLink         docxLink
	httpStatus       int
	errorDescription error
}

func check(docxLink docxLink, channel chan<- linkCheckResult) {
	response, err := http.Get(docxLink.docxUrl)
	if err != nil {
		channel <- linkCheckResult{docxLink: docxLink, errorDescription: err}
	} else {
		channel <- linkCheckResult{docxLink: docxLink, httpStatus: response.StatusCode}
	}
}

// collect collects the links from the input
func collect(in io.Reader) []docxLink {
	scanner := bufio.NewScanner(in)
	var docxLinks []docxLink

	for scanner.Scan() {
		line := strings.Split(scanner.Text(), ",")

		id, err := strconv.Atoi(line[0])
		if err != nil {
			fatal(fmt.Errorf("Error in id format: %v", line[0]))
		}
		url := line[1]
		pageNum, err := strconv.Atoi(line[2])
		if err != nil {
			fatal(fmt.Errorf("Error in page_num number format: %v", line[2]))
		}
		docxLinks = append(docxLinks, docxLink{docxID: id, docxUrl: url, docxPageNum: pageNum})
	}

	if err := scanner.Err(); err != nil {
		fatal(err)
	}

	return docxLinks
}

// outputReport formats the results, sorts by id and writes to a file
func outputReport(linkCheckResults map[int]linkCheckResult) {
	var keys []int
	for k := range linkCheckResults {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	fileHandle, err := os.Create("report.csv")
	if err != nil {
		fatal(err)
	}

	writer := bufio.NewWriter(fileHandle)
	defer fileHandle.Close()
	for _, key := range keys {
		linkCheckResult := linkCheckResults[key]
		lineToOutput := fmt.Sprintf("%v,%v,%v,%v", linkCheckResult.docxLink.docxID, linkCheckResult.docxLink.docxUrl, linkCheckResult.docxLink.docxPageNum, linkCheckResult.httpStatus)

		if linkCheckResult.errorDescription != nil {
			lineToOutput += fmt.Sprintf(",'%v'", linkCheckResult.errorDescription)
		}
		fmt.Fprintln(writer, lineToOutput)
	}
	writer.Flush()
}
