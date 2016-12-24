package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"time"
)

func main() {
	numbPtr := flag.Int("num", 21, "number of file to create with random text")
	amountPtr := flag.Int("amount", 5, "number of max amount")
	startPtr := flag.Bool("start", false, "sentence beginning by «lorem ipsum dolor sit amet...»")
	whatPtr := flag.String("what", "paras", "give number of paras|words|bytes|lists")
	pathFile := flag.String("path", "", "path file")
	flag.Parse()

	numberOfProcess := runtime.NumCPU() * 2

	downloadJobs := make(chan string, *numbPtr)
	textJobs := make(chan []byte, *numbPtr)
	writeJobs := make(chan []byte, *numbPtr)
	done := make(chan bool)

	finishDownload := make(chan bool, numberOfProcess)
	finishText := make(chan bool, numberOfProcess)

	testPath(pathFile)

	go writeFile(pathFile, numbPtr, writeJobs, done)

	for i := 0; i < *numbPtr; i++ {
		generateUrl(amountPtr, startPtr, whatPtr, downloadJobs)
	}
	close(downloadJobs)

	for i := 0; i < numberOfProcess; i++ {
		go downloadUrl(downloadJobs, textJobs, finishDownload)
	}

	for i := 0; i < numberOfProcess; i++ {
		go getLoremText(textJobs, writeJobs, finishText)
	}

	waitingTextJobs(&numberOfProcess, textJobs, finishDownload)
	waitingWriteJobs(&numberOfProcess, writeJobs, finishText)

	<-done
}

func testPath(pathFile *string) {
	if err := notExists(pathFile); err != nil {
		if er := os.MkdirAll(*pathFile, 0777); er != nil {
			panic(er)
		}
	}
}

func notExists(pathFile *string) error {
	_, err := os.Stat(*pathFile)
	if os.IsNotExist(err) {
		return err
	}

	return nil
}

func waitingTextJobs(numberOfProcess *int, textJobs chan []byte, finishDownload chan bool) {
	for {
		if len(finishDownload) == *numberOfProcess {
			close(textJobs)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitingWriteJobs(numberOfProcess *int, writeJobs chan []byte, finishText chan bool) {
	for {
		if len(finishText) == *numberOfProcess {
			close(writeJobs)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func generateUrl(maxAmount *int, startByLorem *bool, what *string, downloadJob chan<- string) {
	// what := "paras"
	var start string
	if *startByLorem {
		start = "yes"
	} else {
		start = "no"
	}

	downloadJob <- fmt.Sprintf("http://www.lipsum.com/feed/xml?amount=%d&what=%s&start=%s", *maxAmount, *what, start)

}

func downloadUrl(downloadJobs <-chan string, textJobs chan<- []byte, finishDownload chan bool) {
	for job := range downloadJobs {
		resp, err := http.Get(job)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		textJobs <- body
	}

	finishDownload <- true
}

func getLoremText(textJobs <-chan []byte, writeJobs chan<- []byte, finishText chan bool) {
	for job := range textJobs {
		begin := bytes.Index(job, []byte("<lipsum>"))
		end := bytes.Index(job, []byte("</lipsum>"))

		writeJobs <- job[begin+8 : end]
	}

	finishText <- true
}

func writeFile(filePath *string, size *int, writeJobs <-chan []byte, done chan bool) {
	num := 0

	for job := range writeJobs {
		beginBytes := []byte(fmt.Sprintf("+++\ndate = 2016-12-23\ndraft = false\ntitle = test%d\nweight = %d\n+++\n\n", num+1, num+1))
		data := append(beginBytes, job...)

		filename := path.Join(*filePath, fmt.Sprintf("File%d.md", num+1))
		if err := ioutil.WriteFile(filename, data, 0666); err != nil {
			panic(err)
		}
		fmt.Printf("«%s» écrit | %d%%\r", filename, (num+1)*100/(*size))
		num++
	}
	fmt.Println()
	done <- true
}
