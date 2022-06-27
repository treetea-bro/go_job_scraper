package scrapper

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type extractedJob struct {
	id string
	title string
	location string
	summary string
}

var baseURL string
func Scrape(term string) {
	baseURL = "https://kr.indeed.com/jobs?q=" + term + "&limit=50"
	var jobs []extractedJob
	c := make(chan []extractedJob)

	totalPages := getPages()
	
	for i := 0; i < totalPages; i++ {
		go getPage(i, c)	
	}

	for i := 0; i < totalPages; i++ {
		extractedJob := <-c
		jobs = append(jobs, extractedJob...)
	}

	writeJobs(jobs)
}

// Scrape Indeed by a term
func getPage(page int, mainC chan<- []extractedJob) {
	var jobs []extractedJob
	c := make(chan extractedJob)
	pageURL := baseURL + "&start=" + strconv.Itoa(page * 50)
	fmt.Println(pageURL)
	res, err := http.Get(pageURL)
	checkErr(err)
	checkCode(res)

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)
	
	searchCards := doc.Find(".cardOutline")

	searchCards.Each(func(i int, s *goquery.Selection) {
		go extractJob(s, c)
	})

	for i := 0; i < searchCards.Length(); i++ {
		jobs = append(jobs, <- c)
	}

	mainC <- jobs
}

func extractJob(s *goquery.Selection, c chan<- extractedJob) {
	id, _ := s.Find("a").Attr("data-jk")
	title := CleanString(s.Find(".jobTitle span").Text())
	location := CleanString(s.Find(".companyLocation").Text())
	summary := CleanString(s.Find(".job-snippet").Text())
	c <- extractedJob {id: id, title: title, location: location, summary: summary}
}

// CleanString cleans a string
func CleanString(str string) string {
	return strings.Join(strings.Fields(str), " ")
}

func getPages() int {
	pages := 0
	res, err := http.Get(baseURL)
	checkErr(err)
	checkCode(res)
	
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	doc.Find(".pagination").Each(func(i int, s *goquery.Selection) {
		pages = s.Find("a").Length()
	})
	return pages
}

func writeJobs(jobs []extractedJob) {
	file, err := os.Create("jobs.csv")
	checkErr(err)

	// utf8bom := []byte{0xEF, 0xBB, 0xBF}
	// file.Write(utf8bom)

	w := csv.NewWriter(file)
	defer w.Flush()
	defer file.Close()

	headers := []string{"Link", "Title", "Location", "Summary"}

	wErr := w.Write(headers)
	checkErr(wErr)

	c := make(chan error)
	for _, job := range jobs {
		go mkJobSlice(w, job, c)
	}

	for i := 0; i < len(jobs); i++ {
		checkErr(<- c)
	}
}

func mkJobSlice(w *csv.Writer, job extractedJob, c chan<- error) {
	jobSlice := []string{"https://kr.indeed.com/viewjob?jk=" + job.id, job.title, job.location, job.summary}
	c <- w.Write(jobSlice)
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkCode(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Request failed with Status:", res.StatusCode)
	}
}