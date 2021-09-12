package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	"gopkg.in/yaml.v3"
)

func init() {
	// log.SetPrefix("")
	log.SetFlags(0)
}

type Command struct {
	Desc             string            `yaml:"desc"`
	Command          []string          `yaml:"command"`
	WorkingDirectory string            `yaml:"workingDirectory"`
	Environment      map[string]string `yaml:"environment"`
	Foreground       bool              `yaml:"foreground"`
	Restart          bool              `yaml:"restart"`

	Aliases []string `yaml:"aliases"`

	process *exec.Cmd
}

func Max(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

var wg sync.WaitGroup

func main() {
	args := os.Args[1:]

	stream, err := ioutil.ReadFile("aliases.yaml")
	if err != nil {
		log.Panic(err.Error())
	}

	var aliases = map[string]Command{}
	yaml.Unmarshal(stream, aliases)

	if len(args) == 0 {
		keys := []string{}
		for k := range aliases {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		max := 0
		for _, key := range keys {
			max = Max(max, len(key))
		}

		format := fmt.Sprintf("%%-%ds: %%s\n", 1+max)
		for _, key := range keys {
			value := aliases[key]
			log.Printf(format, key, value.Desc)
		}

		os.Exit(66)
	}

	name := args[0]
	alias, exists := aliases[name]
	if !exists {
		log.Panicf("undefined alias '%s'", name)
	}

	processes := []Command{}
	if alias.Aliases == nil {
		wg.Add(1)
		callCommand(alias)
		processes = append(processes, alias)
	} else {
		for _, subName := range alias.Aliases {
			subAlias, exists := aliases[subName]
			if !exists {
				log.Panicf("undefined alias '%s'", name)
				continue
			}
			wg.Add(1)
			if alias.Foreground {
				callCommand(subAlias)
			} else {
				go callCommand(subAlias)
			}
			processes = append(processes, subAlias)
		}
	}

	go func() {
		cancelChan := make(chan os.Signal, 1)
		signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)
		sig := <-cancelChan
		log.Printf("Caught SIGTERM %v", sig)
		for _, command := range processes {
			command.Restart = false
			command.process.Process.Signal(syscall.SIGKILL)
		}
	}()

	wg.Wait()
	os.Exit(0)
}

func callCommand(alias Command) {
	for {
		env := os.Environ()
		for key, value := range alias.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}

		command := alias.Command[0]
		tail := alias.Command[:]
		executable, err := exec.LookPath(command)
		if err != nil {
			log.Panic(err.Error())
		}
		tail[0] = executable

		workingDir, _ := os.Getwd()
		if err != nil {
			log.Panic(err.Error())
		}
		workingDir, err = filepath.Abs(filepath.Join(workingDir, alias.WorkingDirectory))
		if err != nil {
			log.Panic(err.Error())
		}

		alias.process = &exec.Cmd{
			Path:   executable,
			Args:   tail,
			Env:    env,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Stdin:  os.Stdin,
			Dir:    workingDir,
		}
		log.Printf("$ '%s' in %+v with %+v %v", strings.Join(alias.process.Args, "' '"), alias.process.Dir, alias.Environment, ab(alias.Restart, "restart", ""))
		alias.process.Start()

		if alias.Foreground || alias.Restart {
			waitProc(alias.process)
			if !alias.Restart {
				break
			}
			alias.process.Process.Kill()
		} else {
			go waitProc(alias.process)
			break
		}
		wg.Add(1)
	}
}

func waitProc(proc *exec.Cmd) {
	defer wg.Done()
	proc.Wait()
}

func ab(dip bool, a string, b string) string {
	if dip {
		return a
	}
	return b
}
