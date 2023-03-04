package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()
	m   = make(map[string]*linkedList)
	mu  *sync.RWMutex
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
		m[queue].addLast(msg)
		mu.Unlock()

		log.Infof("add %s to queue %s", msg, queue)
	case http.MethodGet:
		queue := r.URL.Path

		mu.RLock()
		val, ok := m[queue]
		mu.RUnlock()

		if ok {
			msg := val.head.data
			w.Write([]byte(msg))
			m[queue].deleteFirst()

			if val.head == nil {
				mu.Lock()
				delete(m, queue)
				mu.Unlock()
			}

			log.Infof("get %s from queue %s", msg, queue)
		} else {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}

	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (l *linkedList) addLast(data string) {
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
