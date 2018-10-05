package main

import (
	"fmt"
	"net/http"
	"os"
	"encoding/json"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"runtime"
	"time"
	"runtime/debug"
	"github.com/gorilla/websocket"
	"strconv"
	"flag"
)

type LogFormatter struct{}

func (f *LogFormatter) Format(entry *log.Entry) ([]byte, error) {
	t := entry.Time.Format("2006-01-02T15:04:05.999Z07:00")
	return []byte(fmt.Sprintf("[%s][%s][v1.0.0] %s\n", t, entry.Level.String(), entry.Message)), nil
}

var port = flag.Int("port", 4000, "port")

func init() {
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	log.SetFormatter(&LogFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.Level(5))
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/info", info)
	r.HandleFunc("/", ws)
	http.Handle("/", r)
	log.Debugf("Listen http://0.0.0.0:%d", *port)
	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", *port), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Serve error %v\n", err)
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	health, err := json.Marshal(map[string]interface{}{
		"runtime.NumGoroutine":        runtime.NumGoroutine(),
		"runtime.MemStats.Alloc":      memStats.Alloc,
		"runtime.MemStats.TotalAlloc": memStats.TotalAlloc,
		"runtime.MemStats.Sys":        memStats.Sys,
		"runtime.MemStats.NumGC":      memStats.NumGC,
	})

	if err != nil {
		log.Errorf("Health error: %v", err)
	}
	w.Write(health)
}

func info(w http.ResponseWriter, r *http.Request) {

	gameIds := r.URL.Query()["game"]
	gameId := 0
	var err error
	if len(gameIds) > 0 {
		gameId, err = strconv.Atoi(gameIds[0])
		if err != nil {
			w.Write([]byte("invalid game id"))
		}
	} else {
		w.Write([]byte("param \"game\" can't be empty"))
	}

	game, ok := Games[gameId]
	if !ok {
		w.Write([]byte("invalid game id"))
		return
	}

	playersInfo := make([]interface{}, 0)
	for _, player := range game.Players.FindAll() {
		playersInfo = append(playersInfo, map[string]interface{}{
			"id":        player.Id(),
			"name":      player.Name(),
			"addr":      player.Addr(),
			"createdAt": player.createdAt,
			"role":      player.Role(),
		})
	}

	info := map[string]interface{}{
		"id":           game.Id,
		"event":        game.Event.Name(),
		"event_status": game.Event.Status(),
		"iter":         game.Iteration,
		"win":          game.Winner,
		"is_over":      game.isOver(),
		"players":      playersInfo,
	}

	response, err := json.Marshal(info)

	if err != nil {
		log.Errorf("Info controller error: %v", err)
	}
	w.Write(response)
}

func ws(w http.ResponseWriter, r *http.Request) {

	defer func() {
		log.Infof("CLOSE serverWS")
	}()
	log.Debugf("Server WS")

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Debugf("Upgrade error %s: %v", r.URL.String(), err)
		return
	}

	player := NewPlayer()
	player.SetConnection(conn)
	player.SetAddr(r.RemoteAddr)
}

func GC(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for range t.C {
		debug.FreeOSMemory()
	}
}
