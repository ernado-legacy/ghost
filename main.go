package main

import (
	"fmt"
	// "github.com/GeertJohan/go.rice"
	. "github.com/boltdb/bolt"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	MAX_LATENCY_MS = 2
)

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}
func GetId(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "001")
}

func Realtime(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	conn, err := upgrader.Upgrade(w, r, nil)
	pongs := make(chan time.Time)

	conn.SetPongHandler(func(s string) error {
		log.Println("[realtime]", "pong")
		go func() {
			defer func() {
				e := recover()
				if e != nil {
					log.Println("[realtime]", "timed out pong")
				}
			}()
			pongs <- time.Now()
		}()
		return nil
	})

	if err != nil {
		log.Println("upgrade error", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	log.Println("[realtime]", "connected")
	defer func() {
		conn.Close()
		log.Println("[realtime] closed")
	}()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("[realtime]", "read message error", err)
			return
		}
		if err = conn.WriteMessage(messageType, p); err != nil {
			log.Println("[realtime]", "write message error", err)
			return
		}
		deadline := time.Now().Add(time.Millisecond * MAX_LATENCY_MS)
		if err = conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
			log.Println("[realtime]", "ping error", err)
			return
		}
		go func() {
			now := time.Now()
			select {
			case <-time.After(time.Millisecond * MAX_LATENCY_MS):
				log.Println("[realtime]", "timeout")
				close(pongs)
				conn.Close()
				return
			case t := <-pongs:
				delta := t.Sub(now)
				log.Println("[realtime]", "pinged", delta)
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprint("PING ", delta)))
			}
		}()
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)
	router.GET("/realtime", Realtime)
	// router.GET("/peerjs/:id/:token/id", GetId)
	// router.GET("/peerjs/id", GetId)

	// static files
	// router.Handler("/static", path, http.FileServer(rice.MustFindBox("static").HTTPBox()))
	router.ServeFiles("/static/*filepath", http.Dir("static"))
	db, _ := Open("test", 0666)
	defer os.Remove(db.Path())
	defer db.Close()

	// Start a write transaction.
	db.Update(func(tx *Tx) error {
		// Create a bucket.
		tx.CreateBucket([]byte("widgets"))

		// Set the value "bar" for the key "foo".
		tx.Bucket([]byte("widgets")).Put([]byte("foo"), []byte("bar"))
		return nil
	})

	// Read value back in a different read-only transaction.
	db.View(func(tx *Tx) error {
		value := tx.Bucket([]byte("widgets")).Get([]byte("foo"))
		fmt.Printf("The value of 'foo' is: %s\n", value)
		return nil
	})
	log.Fatal(http.ListenAndServe(":8081", router))
}
