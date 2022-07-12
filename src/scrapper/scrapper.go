package scrapper

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	iconv "github.com/djimenez/iconv-go"

	"github.com/PuerkitoBio/goquery"
)

type extractedJob struct {
	id       string
	title    string
	location string
	salary   string
}

func Scrape(term string) {
	var jobs []extractedJob
	typingBaseUrl(term)
	mainC := make(chan []extractedJob)
	totalPages := getPages(term)
	for i := 0; i < totalPages; i++ {
		go getPage(term, i, mainC)
	}

	for i := 0; i < totalPages; i++ {
		extractedJobs := <-mainC
		jobs = append(jobs, extractedJobs...)
	}

	fmt.Println(jobs)
	writeJobs(jobs)
	fmt.Println("Done. Extracted: ", len(jobs))
}

func typingBaseUrl(term string) string {
	var baseUrl string = "https://search.incruit.com/list/search.asp?col=job&kw=" + term
	return baseUrl
}

func getPage(term string, page int, mainC chan<- []extractedJob) {
	var jobs []extractedJob
	c := make(chan extractedJob)
	baseUrl := typingBaseUrl(term)
	pageUrl := baseUrl + "&startno=" + strconv.Itoa(page*30)
	fmt.Println("Requesting: ", pageUrl)
	res, err := http.Get(pageUrl)
	checkErr(err)
	checkCode(res)

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	searchCard := doc.Find(".c_row").Each(func(i int, card *goquery.Selection) {
		go extractJob(card, c)
	})

	for i := 0; i < searchCard.Length(); i++ {
		job := <-c
		jobs = append(jobs, job)
	}

	mainC <- jobs

}

func extractJob(card *goquery.Selection, c chan<- extractedJob) {
	id, _ := card.Attr("jobno")
	title := card.Find(".cl_top>a").Text()
	out, _ := iconv.ConvertString(string(title), "euc-kr", "utf-8")
	CleanSpace(out)

	location := card.Find(".cl_md span").Text()
	out1, _ := iconv.ConvertString(string(location), "euc-kr", "utf-8")
	CleanSpace(out1)

	c <- extractedJob{
		id:       id,
		title:    out,
		location: out1,
	}

}

func CleanSpace(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}

func getPages(term string) int {
	pages := 0
	baseUrl := typingBaseUrl(term)
	res, err := http.Get(baseUrl)
	checkErr(err)
	checkCode(res)

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	doc.Find(".sqr_paging").Each(func(i int, s *goquery.Selection) {
		pages = s.Find("a").Length() - 1
	})

	return pages
}

func writeJobs(jobs []extractedJob) {
	file, err := os.Create("jobs.csv") //파일생성
	checkErr(err)

	w := csv.NewWriter(file) //연필생성
	defer w.Flush()

	headers := []string{"id", "title", "location"} //헤더생성
	wErr := w.Write(headers)
	checkErr(wErr)

	for _, job := range jobs { //각 정보마다 Write함수로 작성 후 Flush로 저장
		jobSlice := []string{job.id, job.title, job.location}
		jwErr := w.Write(jobSlice)
		checkErr(jwErr)
	}

}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
		fmt.Println(err)
	}
}

func checkCode(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Request failed with statusCode:", res.StatusCode)
	}
}
