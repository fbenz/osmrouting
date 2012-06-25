// Non-blocking logging and timing functionality

package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	LogBufferSize = 1000
)

var (
	loggerChan chan *RequestInfo
)

type RequestInfo struct {
	URL      string
	Time     time.Time
	Duration time.Duration
}

// InitLogger sets up a buffered channel and starts a goroutine so that requests can be logged afterwards.
func InitLogger() {
	loggerChan = make(chan *RequestInfo, LogBufferSize)
	// exactly one logger runs concurrently
	go logger()
}

// LogRequest loggs requests in a non-blocking way. It is guaranteed that the request is logged.
func LogRequest(r *http.Request, startTime time.Time) {
	if !FlagLogging {
		return
	}

	// if the buffer is full, the request is not logged
	if len(loggerChan) < LogBufferSize {
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		loggerChan <- &RequestInfo{r.URL.String(), startTime, duration}
	}
}

// logger processes requests that should be logged, and writes them to disk.
func logger() {
	writeBuffer := make([]*RequestInfo, 0, LogBufferSize)
	// wait until at least one request has to be written
	for currentRequest := range loggerChan {
		// fill the buffer with all pending requests
		for currentRequest != nil {
			writeBuffer = append(writeBuffer, currentRequest)
			select {
			case currentRequest = <-loggerChan:
				// nothing
			default:
				currentRequest = nil
			}
		}

		// open file for appending (file is created if necessary)
		filename := formatLoggerFilename(time.Now())
		file, _ := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		writer := bufio.NewWriter(file)

		for _, ri := range writeBuffer {
			// write a line for each request
			writer.WriteString(ri.Time.Format(time.RFC3339 /* "2006-01-02T15:04:05Z07:00" */))
			writer.WriteString(" ")
			durationInMicroseconds := strconv.FormatInt(ri.Duration.Nanoseconds()/1000, 10 /* base */)
			writer.WriteString(durationInMicroseconds)
			writer.WriteString(" ")
			writer.WriteString(ri.URL)
			writer.WriteString("\n")
		}

		err := writer.Flush()
		if err != nil {
			log.Printf("logger flush error %v\n", err.Error())
		}
		file.Close()

		// reset buffer
		writeBuffer = writeBuffer[:0]
	}
}

// formatLoggerFilename returns the file name of the log file for the given date.
func formatLoggerFilename(t time.Time) string {
	return "log_" + strconv.Itoa(t.Year()) + "_" + strconv.Itoa(int(t.Month())) + "_" + strconv.Itoa(t.Day()) + ".txt"
}
