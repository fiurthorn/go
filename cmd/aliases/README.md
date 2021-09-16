# aliases

Helper for executing and memory of commands.
If it is necessary, it executes commands in parallel.

## yaml structure

```
map[string](Command|SubCommand)
```

### command

```go
type Command struct {
    // displayed along the help site
    Desc             string            `yaml:"desc"`

    // array with command and
    Command          []string          `yaml:"command"`

    // Working directory of the command
    WorkingDirectory string            `yaml:"workingDirectory"`

    // additional environment variable
    Environment      map[string]string `yaml:"environment"`

    // do not run in parallel
    Foreground       bool              `yaml:"foreground"`

    // should the command restartet after it exits
    Restart          bool              `yaml:"restart"`
}
```

### sub-commands

```go
type SubCommand struct {
    // list of [commands]()
    Aliases []string `yaml:"aliases"`
}
```
