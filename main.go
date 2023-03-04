package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()
	m   = make(map[string]*linkedList)
	mu  = &sync.RWMutex{}
)

type linkedList struct {
	head *node
	tail *node
}

type node struct {
	data string
	next *node
}

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Printf("Need just one integer argument(port number)!\n")
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleQueue)

	port := args[1]
	log.Infof("Listening to port %s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("error listening to port: %s", err)
	}
}

func handleQueue(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		msg := r.URL.Query().Get("v")
		if msg == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		queue := r.URL.Path

		mu.Lock()
		if _, ok := m[queue]; !ok {
			m[queue] = &linkedList{}
		}
		m[queue].pushBack(msg)
		mu.Unlock()

		log.Infof("add %s to queue %s", msg, queue)
	case http.MethodGet:
		queue := r.URL.Path
		timeParam := r.URL.Query().Get("timeout")
		if timeParam != "" {
			log.Infof("request with timeout - %s seconds", timeParam)

			timeout, err := strconv.Atoi(timeParam)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}

			msgChan := make(chan string)
			go func() {
				for {
					if msg := firstFromQueue(queue); msg != "" {
						msgChan <- msg
					}
				}
			}()

			// for {
			// 	if msg := firstFromQueue(queue); msg != "" {
			// 		w.Write([]byte(msg))

			// 		log.Infof("get %s from queue %s", msg, queue)

			// 		break
			// 	}
			// }

			select {
			case <-time.After(time.Duration(timeout) * time.Second):
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			case msg := <-msgChan:
				w.Write([]byte(msg))

				log.Infof("get %s from queue %s", msg, queue)
			}

		} else {
			if msg := firstFromQueue(queue); msg != "" {
				w.Write([]byte(msg))

				log.Infof("get %s from queue %s", msg, queue)
			} else {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			}
		}
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func firstFromQueue(queue string) string {
	var msg string
	mu.RLock()
	if _, ok := m[queue]; ok {
		msg = m[queue].firstData()
		m[queue].deleteFirst()
	}
	mu.RUnlock()

	return msg
}

func (l *linkedList) firstData() string {
	if l.head == nil {
		return ""
	}

	return l.head.data
}

func (l *linkedList) pushBack(data string) {
	newNode := &node{
		data: data,
	}

	if l.head == nil {
		l.head = newNode
		l.tail = newNode
	} else {
		prev := l.tail
		l.tail = newNode
		prev.next = l.tail
	}
}

func (l *linkedList) deleteFirst() {
	if l.head == nil {
		return
	}

	newHead := l.head.next
	l.head.next = nil
	l.head = newHead
}
