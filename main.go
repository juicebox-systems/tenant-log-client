package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	endpoint := flag.String("url", "http://localhost:8080", "URL to the tenant event service")
	pageSize := flag.Int("page", 1, "page size")
	token := flag.String("token", "", "Auth token")
	ack := flag.Bool("ack", false, "send ack for received events")
	watch := flag.Bool("watch", false, "continue to poll and watch for events")
	threads := flag.Int("threads", 1, "number of poller threads")

	flag.Parse()
	fmt.Printf("Id, When, Event, NumGuesses,GuessCount, UserId, Ack\n")
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        *threads,
			MaxIdleConnsPerHost: *threads,
			MaxConnsPerHost:     *threads,
		},
	}
	if *watch {
		for i := 0; i < *threads; i++ {
			go func() {
				ackIds := []string{}
				for {
					if !*ack {
						ackIds = ackIds[:0]
					}
					ackIds = pollOnce(c, *endpoint, *token, *pageSize, ackIds)
					if len(ackIds) == 0 {
						time.Sleep(time.Second)
					}
				}
			}()
		}
		select {}
	}
	ackIds := pollOnce(c, *endpoint, *token, *pageSize, nil)
	if *ack {
		sendAcks(c, *endpoint, *token, ackIds)
	}
}

func pollOnce(c *http.Client, endpoint string, token string, pageSize int, ack []string) []string {
	r := Req{
		Ack:      ack,
		PageSize: pageSize,
	}
	body, err := json.Marshal(&r)
	if err != nil {
		log.Fatalf("failed to serialize request: %v", err)
	}
	http_req, err := http.NewRequest("POST", endpoint+"/tenant_log", bytes.NewReader(body))
	if err != nil {
		log.Fatalf("failed to construct http request: %v", err)
	}
	http_req.Header.Add("Authorization", "Bearer "+token)
	res, err := c.Do(http_req)
	if err != nil {
		log.Fatalf("failed to send request: %v", err)
	}
	if res.StatusCode != 200 {
		fmt.Printf("got status code %d\n", res.StatusCode)
		b, _ := io.ReadAll(res.Body)
		log.Fatalf("%s", b)
	}
	b, _ := io.ReadAll(res.Body)
	rlog := RecoveryLog{}
	err = json.Unmarshal(b, &rlog)
	if err != nil {
		log.Fatalf("failed to parse json from server: %v", err)
	}
	ackIds := make([]string, 0, len(rlog.Events))
	for _, e := range rlog.Events {
		fmt.Printf("%v, %v, %-20s,%v, %v, %v, %s\n", e.Id, e.When, e.Event, intVal(e.NumGuesses), intVal(e.GuessCount), e.UserId, e.AckShort())
		ackIds = append(ackIds, e.Ack)
	}
	return ackIds
}

func sendAcks(c *http.Client, endpoint string, token string, ids []string) {
	r := Req{Ack: ids, PageSize: 0}
	body, err := json.Marshal(&r)
	if err != nil {
		log.Fatalf("failed to serialize request: %v", err)
	}
	http_req, err := http.NewRequest("POST", endpoint+"/tenant_log/ack", bytes.NewReader(body))
	if err != nil {
		log.Fatalf("failed to construct http request: %v", err)
	}
	http_req.Header.Add("Authorization", "Bearer "+token)
	res, err := c.Do(http_req)
	if err != nil {
		log.Fatalf("failed to send request: %v", err)
	}
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode == 200 {
		fmt.Printf("Ack'd %d events\n", len(r.Ack))
	} else {
		fmt.Printf("Error ack'ing events: %d: %s\n", res.StatusCode, b)
	}
}

func intVal(pv *uint16) string {
	if pv == nil {
		return " "
	}
	return fmt.Sprintf("%d", *pv)
}

type Req struct {
	Ack      []string `json:"acks"`
	PageSize int      `json:"page_size"`
}

type RecoveryLog struct {
	Events []RecoveryLogEntry `json:"events"`
}

type RecoveryLogEntry struct {
	Id         string    `json:"id"`
	Ack        string    `json:"ack"`
	When       time.Time `json:"when"`
	UserId     string    `json:"user_id"`
	Event      string    `json:"event"`
	NumGuesses *uint16   `json:"num_guesses,omitempty"`
	GuessCount *uint16   `json:"guess_count,omitempty"`
}

func (e *RecoveryLogEntry) AckShort() string {
	if len(e.Ack) < 20 {
		return e.Ack
	}
	return e.Ack[:17] + "..."
}
