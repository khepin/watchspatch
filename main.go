package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsevents"
	"github.com/mb0/glob"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
)

const configFile = ".watchspatch.toml"

func main() {
	path, _ := filepath.Abs(".")
	dev, err := fsevents.DeviceForPath(path)
	if err != nil {
		logrus.Fatalf("Failed to retrieve device for path: %v", err)
	}

	es := &fsevents.EventStream{
		Paths:  []string{path},
		Device: dev,
		Flags:  fsevents.FileEvents | fsevents.WatchRoot,
	}
	es.Start()
	ec := es.Events

	done := make(chan struct{})

	config := Config{}
	config.Reload()

	deb := NewDebouncer()
	go func() {
		for msg := range ec {
			for _, event := range msg {
				abs := "/" + event.Path
				relpath, err := filepath.Rel(path, abs)
				if err != nil {
					logrus.Fatal(err)
				}
				e := FileEvent{
					Event:   event,
					RelPath: relpath,
				}

				if e.RelPath == configFile && !deb.Has(configFile) {
					deb.AddFor(configFile, 200*time.Millisecond)
					config.Reload()
					logrus.Warn("reloaded config")
				}

				matched := false
				for glob, action := range config.Patterns {
					if !match(glob, e.RelPath) {
						continue
					}
					if deb.Has(glob) {
						continue
					}
					deb.AddFor(glob, action.Debounce)
					matched = true
					logrus.WithFields(logrus.Fields{
						"path":    e.RelPath,
						"matched": glob,
						"action":  action.Cmd,
					}).Info()
					cmd := exec.Command("sh", "-c", action.Cmd)
					cmd.Stdin = os.Stdin
					cmd.Stderr = os.Stderr
					cmd.Stdout = os.Stdout
					cmd.Run()
				}
				if !matched {
					logrus.WithFields(logrus.Fields{
						"path":    e.RelPath,
						"matched": nil,
						"action":  nil,
					}).Debug()
				}
			}
		}
	}()

	<-done
}

type Debouncer struct {
	data map[string]struct{}
	mu   *sync.Mutex
}

func NewDebouncer() *Debouncer {
	return &Debouncer{
		data: map[string]struct{}{},
		mu:   &sync.Mutex{},
	}
}

func (d *Debouncer) AddFor(key string, dur time.Duration) {
	defer d.mu.Unlock()
	d.mu.Lock()
	d.data[key] = struct{}{}

	go func(d *Debouncer, key string, dur time.Duration) {
		<-time.After(dur)
		d.Remove(key)
	}(d, key, dur)
}

func (d *Debouncer) Remove(key string) {
	defer d.mu.Unlock()
	d.mu.Lock()
	delete(d.data, key)
}

func (d *Debouncer) Has(key string) bool {
	defer d.mu.Unlock()
	d.mu.Lock()

	_, ok := d.data[key]
	return ok
}

func match(pattern, path string) bool {
	r, err := glob.Match(pattern, path)
	if err != nil {
		return false
	}
	return r
}

type FileEvent struct {
	fsevents.Event
	RelPath string
}

type Config struct {
	Version  int
	Patterns map[string]*Action
	Debounce time.Duration
}

func (c *Config) Reload() {
	cfg, _ := toml.LoadFile(configFile)
	cfg.Unmarshal(c)
	c.Prepare()
}

func (c *Config) Prepare() {
	for _, a := range c.Patterns {
		if a.Debounce == 0 {
			a.Debounce = c.Debounce
		}
	}
	if c.Version != 1 {
		logrus.Fatal("only version 1 for now")
	}
}

type Action struct {
	Cmd      string
	Debounce time.Duration
}
