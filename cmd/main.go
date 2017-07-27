package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	// _ "net/http/pprof"
	// "runtime/pprof"

	"github.com/Tang-RoseChild/chatserver/chat"
	"github.com/gorilla/websocket"

	"github.com/go-redis/redis"
)

var (
	addr         string
	hubName      string
	redisAddr    string
	redisChannel string
	alone        bool
)

// var (
// 	cpuprofile = flag.String("cpuprofile", "", "cpu profile")
// 	memprofile = flag.String("memprofile", "", "cpu profile")
// )

// func profile() {
// 	if *cpuprofile != "" {

// 		f, err := os.Create(*cpuprofile)
// 		if err != nil {
// 			log.Fatal("could not create CPU profile: ", err)
// 		}
// 		if err := pprof.StartCPUProfile(f); err != nil {
// 			log.Fatal("could not start CPU profile: ", err)
// 		}
// 		fmt.Println("start create cpu profile")
// 		defer func() {
// 			pprof.StopCPUProfile()
// 			f.Close()
// 		}()
// 	}

// 	if *memprofile != "" {
// 		f, err := os.Create(*memprofile)

// 		if err != nil {
// 			log.Fatal("could not create memory profile: ", err)
// 		}
// 		runtime.GC() // get up-to-date statistics
// 		if err := pprof.WriteHeapProfile(f); err != nil {
// 			log.Fatal("could not write memory profile: ", err)
// 		}
// 		defer func() {
// 			fmt.Println("write heap profile")
// 			pprof.WriteHeapProfile(f)
// 			f.Close()
// 		}()

// 	}
// }

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.BoolVar(&alone, "alone", true, "stand alone or multi with redis")
	flag.StringVar(&addr, "addr", ":9998", "server addr")
	flag.StringVar(&hubName, "name", "default", "hub name")
	flag.StringVar(&redisAddr, "redis-addr", ":6379", "redis addr")
	flag.StringVar(&redisChannel, "redis-channel", "default", "redis channel")
}

func main() {
	flag.Parse()
	// profile()
	var hub *chat.Hub
	if alone {
		hub = chat.NewHub(hubName, nil, "")
		go (*chat.SingleHub)(hub).Run()
	} else {
		client := redis.NewClient(&redis.Options{
			Addr: redisAddr,
		})
		hub = chat.NewHub(hubName, client, redisChannel)
		go hub.Run()
	}

	http.Handle("/", http.FileServer(http.Dir("./views/")))
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	fmt.Println("listen :: ", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}
}

func serveWs(hub *chat.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, nil, 4096, 4096)
	if err != nil {
		panic(err)
	}

	client := chat.NewClient(hub, conn)
	client.Hub.Join(client)
	go client.Serve()
}

// func serveHome(w http.ResponseWriter, r *http.Request) {
// 	log.Println(r.URL)
// 	if r.URL.Path != "/" {
// 		http.Error(w, "Not found", 404)
// 		return
// 	}
// 	if r.Method != "GET" {
// 		http.Error(w, "Method not allowed", 405)
// 		return
// 	}
// 	// http.ServeFile(w, r, "index.html")
// 	http.FileServer(http.Dir("views"))
// }
