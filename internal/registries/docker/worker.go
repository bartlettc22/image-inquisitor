package docker

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

type WorkerRequest struct {
	url          string
	responseChan chan *WorkerResponse
}

type WorkerResponse struct {
	err       error
	bodyBytes []byte
}

func (r *DockerIORegistry) Run() {

	numWorkers := 5

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		go worker(r.workerRequestChan)
	}
}

func worker(requestChan chan *WorkerRequest) {

	for request := range requestChan {
		for {
			resp, err := http.Get(request.url)
			if err != nil {
				request.responseChan <- &WorkerResponse{
					err: err,
				}
				break
			}
			defer resp.Body.Close()

			// If we're being throttled, wait until the timer resets
			if resp.StatusCode == http.StatusTooManyRequests {
				waitForRateLimiter(resp.Header)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				request.responseChan <- &WorkerResponse{
					err: fmt.Errorf("failed to get tags for '%s': %s", request.url, resp.Status),
				}
				break
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				request.responseChan <- &WorkerResponse{
					err: fmt.Errorf("failed to read http body for '%s': %s", request.url, resp.Status),
				}
				break
			}
			request.responseChan <- &WorkerResponse{
				bodyBytes: bodyBytes,
			}
			break
		}
	}
}

func waitForRateLimiter(header http.Header) {

	// Default if headers not present
	waitFor := time.Duration(60 * time.Second)

	// Spec says use X-Retry-After but that doesn't seem to exist
	if retryAfter, ok := header[http.CanonicalHeaderKey("Retry-After")]; ok {
		if len(retryAfter) >= 1 {
			retryAfterInt, err := strconv.Atoi(retryAfter[0])
			if err != nil {
				log.Warnf("error converting hub.docker.com X-Retry-After header to int: %v", retryAfter[0])
				return
			}
			retryAfterTime := time.Unix(int64(retryAfterInt), 0)
			waitFor = time.Until(retryAfterTime)
		}
	}
	log.Debugf("waiting for hub.docker.com rate limit in %s seconds", waitFor.String())
	time.Sleep(waitFor)
}
