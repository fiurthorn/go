package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	colorGreen = string([]byte{27, 91, 57, 55, 59, 51, 50, 59, 49, 109})
	colorRed   = string([]byte{27, 91, 57, 55, 59, 51, 49, 59, 49, 109})
	colorReset = string([]byte{27, 91, 48, 109})

	Watch      []string
	Executable string
	MainEntry  string
	WatchDirs  WatchDirSlice

	stopped bool
	built   = make(chan bool, 1)
	restart = make(chan bool, 1)
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("[builder] ")

	flag.StringVar(&MainEntry, "main", "cmd/main/main.go", "go main function file")
	flag.StringVar(&Executable, "exec", "main", "the built executable")
	flag.Func("watch", "watch base directories", func(p string) error {
		Watch = append(Watch, p)
		return nil
	})
	flag.Parse()
}

func signalHandler() {
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)
	sig := <-cancelChan
	log.Printf("Caught signal %v", sig)

	stopped = true
	close(restart)
	close(built)
	wg.Done()
}

var wg sync.WaitGroup

func main() {
	go signalHandler()

	wg.Add(1)

	for _, dir := range Watch {
		filepath.WalkDir(dir, walker)
	}

	go restarter()
	go builder()
	built <- true

	go watcher()

	proxy()

	wg.Wait()
	time.Sleep(1 * time.Second)
}

func restarter() {
	select {
	case <-restart:
	}

	workingDir, err := os.Getwd()
	if err != nil {
		log.Panic(err.Error())
	}

	for !stopped {
		process := &exec.Cmd{
			Path:   Executable,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Args:   []string{Executable},
			Dir:    workingDir,
		}
		err := process.Start()
		if err != nil {
			log.Printf("%s%v%s", colorRed, err, colorReset)
		}

		select {
		case <-restart:
		}
		process.Process.Kill()
	}
}

func builder() {
	for range built {
		modTimes.Map[Executable] = time.Now().Unix()

		executable, err := exec.LookPath("go")
		if err != nil {
			log.Panic(err.Error())
		}

		workingDir, err := os.Getwd()
		if err != nil {
			log.Panic(err.Error())
		}

		process := &exec.Cmd{
			Path:   executable,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Args:   []string{"go", "build", MainEntry},
			Dir:    workingDir,
		}
		err = process.Start()
		process.Wait()
		if err != nil {
			log.Printf("%s%v%s", colorRed, err, colorReset)
		} else if !process.ProcessState.Success() {
			log.Printf("%sbuild error%s", colorRed, colorReset)
		} else {
			restart <- true
		}
	}
}

func walker(path string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if entry != nil && entry.IsDir() {
		WatchDirs.Add(path)
	}
	return nil
}

var modTimes = ModTimestamps{Map: map[string]int64{}}

func watcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if (event.Op & (fsnotify.Remove)) > 0 {
					log.Printf("unwatch %s", event.Name)
					watcher.Remove(event.Name)
					WatchDirs.Remove(event.Name)
				} else if (event.Op & (fsnotify.Write | fsnotify.Create)) > 0 {
					file, _ := os.Stat(event.Name)
					if file.IsDir() {
						if !WatchDirs.Contains(event.Name) {
							log.Printf("watch %s", event.Name)
							WatchDirs.Add(event.Name)
							err = watcher.Add(event.Name)
						}
						continue
					}

					modTime := file.ModTime().Unix()
					if modTimes.After(event.Name, modTime) {
						modTimes.Map[event.Name] = modTime
						buildTime := modTimes.Map[Executable]

						if math.Abs(float64(modTime-buildTime)) > 1 {
							built <- true
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	for _, dir := range WatchDirs.Data {
		log.Printf("watch %s", dir)
		err = watcher.Add(dir)
	}
	if err != nil {
		return err
	}
	wg.Wait()

	log.Println("close watcher")
	return nil
}

type WatchDirSlice struct {
	Data sort.StringSlice
}

func (s *WatchDirSlice) Contains(value string) bool {
	idx := s.Data.Search(value)
	return idx < len(s.Data) && s.Data[idx] == value
}

func (s *WatchDirSlice) Add(value string) {
	if !s.Contains(value) {
		s.Data = append(s.Data, value)
		s.Data.Sort()
	}
}

func (s *WatchDirSlice) Remove(value string) {
	idx := s.Data.Search(value)
	length := len(s.Data)
	if idx < length && s.Data[idx] == value {
		s.Data[idx] = s.Data[length-1]
		s.Data = s.Data[:length-1]
		s.Data.Sort()
	}
}

type ModTimestamps struct {
	Map  map[string]int64
	lock sync.Mutex
}

func (m *ModTimestamps) After(name string, timestamp int64) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	current := m.Map[name]
	if timestamp > current {
		m.Map[name] = timestamp
		return true
	}

	return false
}

func proxy() {
	port := os.Getenv("PORT")
	proxyPort := os.Getenv("PROXYPORT")

	if len(port) > 0 && len(proxyPort) > 0 {
		(&Proxy{}).Run(port, proxyPort)
	}
}

type Proxy struct {
	listener net.Listener
}

func (p *Proxy) Run(port, proxyPort string) error {
	url, err := url.Parse(fmt.Sprintf("http://localhost:%s", port))
	if err != nil {
		return err
	}
	proxy := httputil.NewSingleHostReverseProxy(url)

	server := http.Server{Handler: http.HandlerFunc(proxy.ServeHTTP)}

	p.listener, err = net.Listen("tcp", fmt.Sprintf("localhost:%s", proxyPort))
	if err != nil {
		return err
	}

	go server.Serve(p.listener)

	return nil
}

func (p *Proxy) Close() error {
	return p.listener.Close()
}
