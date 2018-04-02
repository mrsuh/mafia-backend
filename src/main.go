package main

import (
	"fmt"
	"net/http"
	"os"
	"encoding/json"
	"github.com/gorilla/mux"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"
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

var configPath = flag.String("conf", "./../app/config.yml", "path to config")

func init() {
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	config.SetConfigFile(*configPath)
	err := config.ReadInConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Read config file: %v \n", err)
		os.Exit(1)
	}
	f, err := os.OpenFile(config.GetString("log.file"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Open log file error %v\n", err)
		os.Exit(1)
	}
	log.SetFormatter(&LogFormatter{})
	log.SetOutput(f)
	log.SetLevel(log.Level(config.GetInt("log.level")))
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { health(w, r) })
	r.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) { clients(w, r) })
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { ws(w, r) })
	http.Handle("/", r)

	err := http.ListenAndServe(config.GetString("server.listen"), nil)
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

func clients(w http.ResponseWriter, r *http.Request) {

	gameIds := r.URL.Query()["game"]
	gameId := 0
	var err error
	if len(gameIds) > 0 {
		gameId, err = strconv.Atoi(gameIds[0])
		if err != nil {
			//todo
		}
	}

	game, ok := Games[gameId]

	if !ok {
		//todo
	}

	info := make([]interface{}, 0)
	for _, player := range game.Players.FindAll() {
		info = append(info, map[string]interface{}{
			"id":        player.Id(),
			//"id":        player.Id,
			//"name":      player.Name,
			//"addr":      player.Addr,
			//"version":   player.Version,
			//"url":       player.Url,
			//"device":    player.Device,
			//"createdAt": player.CreatedAt,
		})
	}

	response, err := json.Marshal(info)

	if err != nil {
		log.Errorf("Clients controller error: %v", err)
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
}

func GC(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for range t.C {
		debug.FreeOSMemory()
	}
}
