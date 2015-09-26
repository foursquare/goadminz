package adminz

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/theevocater/go-atomicbool"
)

var running = atomicbool.New()

var killfileTicker = time.NewTicker(time.Second)

var port string

var servicez func() string

func killfile() bool {
	file, err := os.Open("/dev/shm/healthz/kill.all")
	if file != nil && err == nil {
		file.Close()
		return true
	}
	file, err = os.Open(fmt.Sprintf("/dev/shm/healthz/kill.%s", port))
	if file != nil && err == nil {
		file.Close()
		return true
	}
	return false
}

func killfileLoop() {
	for _ = range killfileTicker.C {
		running.Set(!killfile())
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if running.Get() {
		w.WriteHeader(http.StatusOK)
		w.Write(([]byte)("OK"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write(([]byte)("Service Unavailable"))
	}
}

func servicezHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/json")
	w.WriteHeader(http.StatusOK)
	// TODO I probably need to serialize reads to servicez as who knows what
	// people will put in that function
	w.Write(([]byte)(servicez()))
}

// I don't love the way this is init'd
func Init(p string, s func() string) {
	port = p
	servicez = s

	go killfileLoop()

	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/servicez", servicezHandler)
	log.Print("adminz registered")
}

func Stop() {
	killfileTicker.Stop()
	running.Set(false)
}
