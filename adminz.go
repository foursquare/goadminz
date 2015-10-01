// Package adminz provides a simple set of adminz pages for administering
// a simple go server.
package adminz

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/theevocater/go-atomicbool"
)

type Adminz struct {
	// keep track of killfile state
	Killed *atomicbool.AtomicBool

	// ticker that checks killfiles every 1 second
	killfileTicker *time.Ticker

	// list of killfilePaths to check
	killfilePaths []string

	// generates data to return to /servicez endpoint. marshalled into json.
	servicez func() interface{}

	// resume is called when the server is unkilled
	resume func() error

	// pause is called when the server is killed
	pause func() error

	// healthy returns true iff the server is ready to respond to requests
	healthy func() bool
}

func New() *Adminz {
	return &Adminz{
		killfileTicker: time.NewTicker(time.Second),
		Killed:         atomicbool.New(),
	}
}

// resume is called when the server is unkilled
func (a *Adminz) Resume(resume func() error) *Adminz {
	a.resume = resume
	return a
}

// pause is called when the server is killed
func (a *Adminz) Pause(pause func() error) *Adminz {
	a.pause = pause
	return a
}

// healthy returns true iff the server is ready to respond to requests
func (a *Adminz) Healthy(healthy func() bool) *Adminz {
	a.healthy = healthy
	return a
}

// servicez generates data to return to /servicez endpoint. marshalled into
// json.
func (a *Adminz) Servicez(servicez func() interface{}) *Adminz {
	a.servicez = servicez
	return a
}

// list of killfilePaths to check
func (a *Adminz) KillfilePaths(killfilePaths []string) *Adminz {
	a.killfilePaths = killfilePaths
	return a
}

func (a *Adminz) Build() *Adminz {
	// start killfile checking loop
	if len(a.killfilePaths) != 0 {
		go a.killfileLoop()
	} else {
		log.Print("Not checking killfiles.")
	}

	http.HandleFunc("/healthz", a.healthzHandler)
	http.HandleFunc("/servicez", a.servicezHandler)

	log.Print("adminz registered")
	log.Print("Watching paths for killfile: ", a.killfilePaths)
	return a
}

// Generates the standard set of killfiles. Pass these to Init()
func Killfiles(ports ...string) []string {
	// the number of ports + the "all" killfile
	var ret = make([]string, len(ports)+1)
	for i, port := range ports {
		ret[i] = fmt.Sprintf("/dev/shm/healthz/kill.%s", port)
	}
	ret[len(ports)] = "/dev/shm/healthz/kill.all"
	return ret
}

func (a *Adminz) checkKillfiles() bool {
	for _, killfile := range a.killfilePaths {
		file, err := os.Open(killfile)
		if file != nil && err == nil {
			file.Close()
			return true
		}
	}
	return false
}

func (a *Adminz) killfileLoop() {
	for _ = range a.killfileTicker.C {
		current := a.Killed.Get()
		next := a.checkKillfiles()
		if current == false && next == true {
			// If we are currently not running and the killfile is removed, call resume()
			a.resume()
			a.Killed.Set(next)
		} else if current == true && next == false {
			// If we are currently running and a killfile is dropped, call pause()
			a.pause()
			a.Killed.Set(next)
		}
		// If we hit neither of those, no state changed.
	}
}

func (a *Adminz) healthzHandler(w http.ResponseWriter, r *http.Request) {
	// we are healthy iff:
	// we are not killed AND
	// a.healthy is unset (so we ignore it) OR
	// a.healthy() returns true
	var ret string
	if !a.Killed.Get() && (a.healthy == nil || a.healthy()) {
		ret = "OK"
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		ret = "Service Unavailable"
	}
	log.Print("Healthz returning ", ret)
	w.Write(([]byte)(ret))
}

func (a *Adminz) servicezHandler(w http.ResponseWriter, r *http.Request) {
	if a.servicez == nil {
		return
	}

	bytes, err := json.Marshal(a.servicez())
	if err == nil {
		w.Header().Add("Content-Type", "text/json")
		// TODO I probably need to serialize reads to servicez as who knows what
		// people will put in that function
		w.Write(bytes)
	} else {
		http.Error(w, err.Error(), 500)
	}
}
