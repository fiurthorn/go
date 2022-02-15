package main

import (
	"errors"
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
	"time"

	_ "embed"

	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

func init() {
	// log.SetPrefix("")
	log.SetFlags(0)
}

//go:embed aliases.yaml
var template []byte

type Command struct {
	Name             string
	Desc             string            `yaml:"desc"`
	Command          string            `yaml:"command"`
	Args             string            `yaml:"args"`
	Array            []string          `yaml:"argsArray"`
	WorkingDirectory string            `yaml:"workingDirectory"`
	Environment      map[string]string `yaml:"environment"`
	Background       bool              `yaml:"background"`
	Restart          bool              `yaml:"restart"`
	Shortcut         string            `yaml:"shortcut"`

	Aliases []string `yaml:"aliases"`

	process *exec.Cmd
	done    bool
}

func (c *Command) Done() {
	c.done = true
}

func Max(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

var (
	wg       sync.WaitGroup
	stopped  bool
	yamlFile string = "aliases.yaml"
)

func main() {
	if _, err := os.Stat(yamlFile); errors.Is(err, os.ErrNotExist) {
		app := tview.NewApplication()
		list := tview.NewList()
		list.SetBorder(true)
		list.SetTitle("Select an item")
		list.AddItem("Example", "Write example file?", 'y', func() { ioutil.WriteFile(yamlFile, template, 0666); app.Stop() })
		list.AddItem("Quit", "Press to exit", 'q', func() { app.Stop() })
		if err := app.SetRoot(list, true).EnableMouse(true).Run(); err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	args := os.Args[1:]

	stream, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		log.Panic(err.Error())
	}

	var aliases = map[string]Command{}
	yaml.Unmarshal(stream, aliases)

	cmd, filter := getFilter(args)

	if filter {
		keys := []string{}
		for k, v := range aliases {
			if len(v.Desc) > 0 {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)

		app := tview.NewApplication()
		list := tview.NewList()
		list.SetBorder(true)
		list.SetTitle("Select an item")
		for _, key := range keys {
			if strings.HasPrefix(key, cmd) {
				name := key
				alias := aliases[name]
				list.AddItem(name, alias.Desc, shortcut(alias.Shortcut), func() { app.Stop(); fmt.Println("$ aliases", name); callAlias(name, alias, aliases) })
			}
		}
		list.AddItem("Quit", "Press to exit", 'q', func() { app.Stop() })
		if err := app.SetRoot(list, true).EnableMouse(true).Run(); err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	alias, exists := aliases[cmd]
	if !exists {
		describe(aliases)
	}

	callAlias(cmd, alias, aliases)
}

func getFilter(args []string) (string, bool) {
	if len(args) == 0 {
		return "", true
	}

	name := args[0]
	filter := name[len(name)-1] == '*'
	if filter {
		name = name[0 : len(name)-1]
	}

	return name, filter

}

func describe(aliases map[string]Command) {
	keys := []string{}
	for k, v := range aliases {
		if len(v.Desc) > 0 {
			keys = append(keys, k)
		}
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

func shortcut(value string) rune {
	if len(value) > 0 {
		return []rune(value)[0]
	}

	return ' '
}

func checkAlive() bool {
	for _, command := range processes {
		if !command.done {
			return true
		}
	}
	return false
}

func sendSignal(signal syscall.Signal) {
	if checkAlive() {
		for _, command := range processes {
			command.Restart = false
			if command.process != nil && !command.done {
				log.Printf("send %s to '%v'", signal.String(), command.Name)
				err := command.process.Process.Signal(signal)
				if err != nil {
					log.Print("int", err.Error())
				}
			}
		}
	}
}

func signalHandler() {
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)
	sig := <-cancelChan
	wg.Add(1)
	stopped = true
	log.Printf("Caught signal %v for %d processes", sig, len(processes))

	sendSignal(syscall.SIGTERM)
	log.Printf("Wait 5 Sec to SIGKILL")
	time.Sleep(5 * time.Second)
	sendSignal(syscall.SIGKILL)

	log.Printf("done")

	wg.Done()
}

var processes = []*Command{}

func callAlias(name string, alias Command, aliases map[string]Command) {
	log.SetPrefix("[aliases] ")

	alias.Name = name

	go signalHandler()

	if alias.Aliases == nil {
		wg.Add(1)
		processes = append(processes, &alias)
		callCommand(&alias)
	} else {
		for _, subName := range alias.Aliases {
			_, exists := aliases[subName]
			if !exists {
				log.Panicf("undefined sub-alias '%s'", subName)
				continue
			}
		}
		for _, subName := range alias.Aliases {
			if !stopped {
				subAlias, _ := aliases[subName]
				subAlias.Name = subName

				processes = append(processes, &subAlias)
				wg.Add(1)

				if subAlias.Background {
					go callCommand(&subAlias)
				} else {
					callCommand(&subAlias)
				}
			}
		}
	}

	wg.Wait()
	os.Exit(0)
}

func buildCommandAndArgs(alias *Command) (string, []string) {
	executable, err := exec.LookPath(alias.Command)
	if err != nil {
		log.Panic(err.Error())
	}
	tail := []string{executable}

	var quote rune = 0
	word := strings.Builder{}

	for _, char := range alias.Args {
		if quote != 0 && char == quote {
			quote = 0
		} else if (char == '"' || char == '\'') && quote == 0 {
			quote = char
		} else if char == ' ' && word.Len() > 0 && quote == 0 {
			tail = append(tail, word.String())
			word.Reset()
		} else {
			word.WriteRune(char)
		}
	}

	if word.Len() > 0 {
		tail = append(tail, word.String())
	}

	if len(alias.Array) > 0 {
		tail = append(tail, alias.Array...)
	}

	return executable, tail
}

func callCommand(alias *Command) {
	for {
		env := os.Environ()
		for key, value := range alias.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}

		executable, args := buildCommandAndArgs(alias)
		workingDir, err := os.Getwd()
		if err != nil {
			log.Panic(err.Error())
		}
		workingDir, err = filepath.Abs(filepath.Join(workingDir, alias.WorkingDirectory))
		if err != nil {
			log.Panic(err.Error())
		}

		alias.process = &exec.Cmd{
			Path:   executable,
			Args:   args,
			Env:    env,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Dir:    workingDir,
		}

		executable = strings.ToLower(executable)
		if !(strings.HasSuffix(executable, "cmd") || strings.HasSuffix(executable, "bat")) {
			alias.process.Stdin = os.Stdin
		} else {
			log.Printf("don't inherit Stdin to %s", executable)
		}
		log.Printf("$ '%s' in %+v with %+v %v [%v]", strings.Join(alias.process.Args, "' '"), alias.process.Dir, alias.Environment, ab(alias.Restart && !stopped, "restart", ""), ab(alias.Background, "bg", "fg"))
		alias.process.Start()

		if !alias.Background || (alias.Restart && !stopped) {
			waitProc(alias)
			if stopped || !alias.Restart {
				break
			}
			if alias.process != nil && alias.process.ProcessState != nil && !alias.process.ProcessState.Exited() {
				alias.process.Process.Kill()
			}
		} else {
			go waitProc(alias)
			break
		}
		wg.Add(1)
	}
}

func waitProc(cmd *Command) {
	defer wg.Done()
	defer cmd.Done()
	cmd.process.Wait()
}

func ab(dip bool, a string, b string) string {
	if dip {
		return a
	}
	return b
}
