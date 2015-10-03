package adminz

import (
	//"bytes"
	//"fmt"
	//"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ExampleAdminz_Build() {
	// To set up the adminz pages, first call New, then add whichever handlers
	// you need, then call build.
	a := New()
	a.Pause(func() error { /* do a thing */ return nil })
	a.Resume(func() error { /* do a thing */ return nil })
	a.Servicez(func() interface{} { return "{}" })
	a.Healthy(func() bool { return true })
	// If you don't add KillfilePaths, there will be no killfile checking.
	a.KillfilePaths(Killfiles("4000"))
	a.Build()
}

func TestKillfilePaths(t *testing.T) {
	killfile := path.Join(os.TempDir(), "kill")
	a := New()
	a.KillfilePaths([]string{killfile})
	a.Build()
	defer a.Stop()

	assert.False(t, a.Killed.Get(), "Killfile shouldn't exist")
	k, err := os.Create(killfile)
	assert.Nil(t, err, "Unable to create killfile")
	defer os.Remove(killfile)
	defer k.Close()

	// Sleep for 2 seconds to ensure the ticker has run
	time.Sleep(time.Second * 2)
	assert.True(t, a.Killed.Get(), "Killfile missed")
}

// Can't run this until I figure out how to tear up and down http stuff.
// Otherwise I reregister the handlers.
//func TestBuildNoInputs(t *testing.T) {
//a := New()
//a.Build()
//defer a.Stop()
//}
