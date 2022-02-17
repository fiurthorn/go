package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"

	"github.com/fsnotify/fsnotify"
)

type Field struct {
	Name string `json:"name"`

	Dart          string `json:"dart"`
	Go            string `json:"go"`
	Type          string `json:"type"`
	GoType        string `json:"goType"`
	DartIfaceType string `json:"dartIfaceType"`
	DartImplType  string `json:"dartImplType"`
	FromJson      string `json:"fromJson"`
	ToJson        string `json:"toJson"`
}

var (
	colorGreen = string([]byte{27, 91, 57, 55, 59, 51, 50, 59, 49, 109})
	colorRed   = string([]byte{27, 91, 57, 55, 59, 51, 49, 59, 49, 109})
	colorReset = string([]byte{27, 91, 48, 109})
)

type Model struct {
	Meta struct {
		Inherit string `json:"inherit"`

		Go struct {
			Package  string   `json:"package"`
			Struct   string   `json:"struct"`
			Import   []string `json:"import"`
			Filename string   `json:"filename"`

			Fields []Field
		} `json:"go"`

		Dart struct {
			Implementation struct {
				Class    string   `json:"class"`
				Import   []string `json:"import"`
				Filename string   `json:"filename"`

				Interface string
				Fields    []Field
			} `json:"implementation"`

			Interface struct {
				Class    string   `json:"class"`
				Import   []string `json:"import"`
				Filename string   `json:"filename"`
				Package  string   `json:"package"`

				Fields []Field
			} `json:"interface"`
		} `json:"dart"`
	} `json:"meta"`

	Fields []Field `json:"fields"`
}

var inheritance = Inheritance{Map: map[string]sort.StringSlice{}}

func NewModel(path string) (*Model, error) {
	tag, _ := filepath.Rel(ModelDir, path)
	log.Println("readSchema", tag)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("error read file: %v", err)
		return nil, err
	}

	schema := &Model{}
	json.Unmarshal(bytes, schema)

	if len(schema.Meta.Inherit) > 0 {
		to, _ := filepath.Abs(path)
		from, _ := filepath.Abs(filepath.Join(filepath.Dir(to), schema.Meta.Inherit))

		inheritance.Add(from, to)
	}

	return schema, nil
}

func (m *Model) AdjustData(path string) error {
	err := m.loadFieldData(filepath.Dir(path), m.Meta.Inherit, m.Fields)
	if err != nil {
		return err
	}
	m.Meta.Dart.Implementation.Interface = m.Meta.Dart.Interface.Class
	m.Meta.Dart.Implementation.Import = append(m.Meta.Dart.Implementation.Import, m.Meta.Dart.Interface.Import...)

	for i, length := 0, len(m.Meta.Dart.Interface.Import); i < length; i++ {
		m.Meta.Dart.Interface.Import[i] = fmt.Sprintf("package:%s/%s", m.Meta.Dart.Interface.Package, m.Meta.Dart.Interface.Import[i])
	}

	ifaceFilePath, _ := filepath.Rel(BaseDartDir, filepath.Join(BaseDir, m.Meta.Dart.Interface.Filename))
	m.Meta.Dart.Implementation.Import = append(m.Meta.Dart.Implementation.Import, filepath.ToSlash(ifaceFilePath))
	for i, length := 0, len(m.Meta.Dart.Implementation.Import); i < length; i++ {
		m.Meta.Dart.Implementation.Import[i] = fmt.Sprintf("package:%s/%s", m.Meta.Dart.Interface.Package, m.Meta.Dart.Implementation.Import[i])
	}

	return nil
}

func (m *Model) loadFieldData(basepath string, inherit string, fields []Field) error {
	if len(inherit) > 0 {
		inherited, err := NewModel(filepath.Join(basepath, inherit))
		if err != nil {
			return err
		}
		err = m.loadFieldData(basepath, inherited.Meta.Inherit, inherited.Fields)
		if err != nil {
			return err
		}
	}

	for _, field := range fields {
		m.Meta.Go.Fields = append(m.Meta.Go.Fields, field)
		m.Meta.Dart.Interface.Fields = append(m.Meta.Dart.Interface.Fields, field)
		m.Meta.Dart.Implementation.Fields = append(m.Meta.Dart.Implementation.Fields, field)
	}

	return nil
}

var (
	ModelDir    string
	BaseDir     string
	BaseDartDir string
	WatchMode   bool
	WatchDirs   WatchDirSlice
)

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

func init() {
	log.SetFlags(0)
	log.SetPrefix("[models] ")

	flag.BoolVar(&WatchMode, "watch", false, "watch mode")
	flag.StringVar(&BaseDir, "base", "", "base directory")
	flag.StringVar(&BaseDartDir, "dart", "", "base dart lib directory")
	flag.StringVar(&ModelDir, "models", "", "models base directory")
	flag.Parse()

	if len(BaseDir) == 0 {
		panic("needed -base is unset")
	}

	if len(ModelDir) == 0 {
		ModelDir = filepath.Join(BaseDir, "models")
	}

	if len(BaseDartDir) == 0 {
		BaseDartDir = filepath.Join(BaseDir, "client", "lib")
	}
}

var wg sync.WaitGroup

func main() {
	err := initTemplates()
	if err != nil {
		panic(err)
	}

	filepath.WalkDir(ModelDir, walker)
	wg.Wait()

	if WatchMode {
		err = watcher()
		if err != nil {
			panic(err)
		}
	}
}

func walker(path string, entry fs.DirEntry, err error) error {
	if entry.IsDir() {
		WatchDirs.Add(path)
		return nil
	}

	if err != nil {
		return err
	}

	if filepath.Ext(path) != ".json" {
		return nil
	}

	wg.Add(1)
	go func() {
		err := evaluate(path, false)
		if err != nil {
			log.Printf("evaluate %s:%v", filepath.Base(path), err)
		}
		wg.Done()
	}()
	return nil
}

func watcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	modTimes := ModTimestamps{Map: map[string]int64{}}

	done := make(chan bool)
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
					modTime := file.ModTime().Unix()

					if file.IsDir() {
						if !WatchDirs.Contains(event.Name) {
							log.Printf("watch %s", event.Name)
							WatchDirs.Add(event.Name)
							err = watcher.Add(event.Name)
						}
						continue
					}

					if modTimes.After(event.Name, modTime) {
						modTimes.Map[event.Name] = modTime

						time.Sleep(50 * time.Millisecond)
						err := evaluate(event.Name, true)
						if err != nil {
							log.Printf("written %s:%v", event.Name, err)
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
	<-done

	return nil
}

type ModelTemplate struct {
	model *template.Template
}

func (m *ModelTemplate) EvalGo(outPath string, data interface{}) error {
	content := &bytes.Buffer{}
	err := m.EvalWriter(content, data)
	if err != nil {
		return err
	}
	formatted, err := format.Source(content.Bytes())
	if err != nil {
		return err
	}

	os.MkdirAll(filepath.Dir(outPath), 0777)

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	io.Copy(out, bytes.NewBuffer(formatted))
	if err != nil {
		return err
	}

	return nil

}

func (m *ModelTemplate) Eval(outPath string, data interface{}) error {
	os.MkdirAll(filepath.Dir(outPath), 0777)

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	err = m.EvalWriter(out, data)
	if err != nil {
		return err
	}

	return nil

}

func (m *ModelTemplate) EvalWriter(out io.Writer, data interface{}) error {
	err := m.model.Execute(out, data)
	if err != nil {
		return err
	}

	return nil

}

//go:embed tmpl/struct.go.tmpl
var goTemplateData string

// string, float64, bool, time.Time, []string
var (
	TypeString   = "String"
	TypeFloat64  = "Float64"
	TypeInt64    = "Int64"
	TypeBool     = "Bool"
	TypeDateTime = "DateTime"
	TypeArray    = "Array"
)

func LoadGoTemplate() (*ModelTemplate, error) {
	var typeMap = map[string]string{
		TypeString:   "string",
		TypeFloat64:  "float64",
		TypeInt64:    "int64",
		TypeBool:     "bool{}",
		TypeDateTime: "*CustomTime",
		TypeArray:    "[]string",
	}

	camelCase := func(name string) string {
		target := []rune(name[:])
		target[0] = unicode.ToUpper(target[0])
		return string(target)
	}

	types := func(key string) string {
		value, ok := typeMap[key]
		if ok {
			return value
		}
		return key
	}

	m, err := template.New("go").
		Funcs(map[string]interface{}{
			"goType":    types,
			"camelCase": camelCase,
		}).
		Parse(goTemplateData)

	return &ModelTemplate{m}, err
}

//go:embed tmpl/iface.dart.tmpl
var dartIfaceData string

//go:embed tmpl/impl.dart.tmpl
var dartImplData string

func LoadDartTemplates() (impl *ModelTemplate, iface *ModelTemplate, err error) {
	var typeMap = map[string]string{
		TypeString:   "String",
		TypeFloat64:  "double",
		TypeInt64:    "int",
		TypeBool:     "bool",
		TypeDateTime: "DateTime",
		TypeArray:    "List<String>",
	}

	types := func(key string) string {
		value, ok := typeMap[key]
		if ok {
			return value
		}
		return key
	}

	baseGenName := func(filename string) string {
		return strings.TrimSuffix(filepath.Base(filename), ".dart")
	}

	snakeCase := func(name string) string {
		target := []rune(name[:])
		target[0] = unicode.ToLower(target[0])
		return string(target)
	}

	var tmpl *template.Template
	tmpl, err = template.New("dart_impl").
		Funcs(map[string]interface{}{
			"dartType":  types,
			"baseGName": baseGenName,
			"snakeCase": snakeCase,
		}).
		Parse(dartImplData)
	if err != nil {
		return
	}
	impl = &ModelTemplate{tmpl}

	tmpl, err = template.New("dart_iface").
		Funcs(map[string]interface{}{
			"dartType":  types,
			"snakeCase": snakeCase,
		}).
		Parse(dartIfaceData)
	if err != nil {
		return
	}
	iface = &ModelTemplate{tmpl}

	return
}

var (
	goTemplate        *ModelTemplate
	dartIfaceTemplate *ModelTemplate
	dartImplTemplate  *ModelTemplate
)

func initTemplates() (err error) {
	goTemplate, err = LoadGoTemplate()
	if err != nil {
		return
	}

	dartImplTemplate, dartIfaceTemplate, err = LoadDartTemplates()
	if err != nil {
		return
	}

	return
}

func evaluate(path string, inherit bool) error {
	tag, _ := filepath.Rel(ModelDir, strings.TrimSuffix(path, ".json"))
	log.Println("evaluate model", tag)

	schema, err := NewModel(path)
	if err != nil {
		return err
	}

	err = schema.AdjustData(path)
	if err != nil {
		return err
	}

	if len(schema.Meta.Go.Filename) > 0 {
		log.Println("emit go", tag, schema.Meta.Go.Struct)
		err = goTemplate.EvalGo(filepath.Join(BaseDir, schema.Meta.Go.Filename), schema.Meta.Go)
		if err != nil {
			return err
		}
	}

	if len(schema.Meta.Dart.Interface.Filename) > 0 {
		log.Println("emit dart interface", tag, schema.Meta.Dart.Interface.Class)
		err = dartIfaceTemplate.Eval(filepath.Join(BaseDir, schema.Meta.Dart.Interface.Filename), schema.Meta.Dart.Interface)
		if err != nil {
			return err
		}
	}

	if len(schema.Meta.Dart.Implementation.Filename) > 0 {
		log.Println("emit dart implementation", tag, schema.Meta.Dart.Implementation.Class)
		err = dartImplTemplate.Eval(filepath.Join(BaseDir, schema.Meta.Dart.Implementation.Filename), schema.Meta.Dart.Implementation)
		if err != nil {
			return err
		}
	}

	if inherit {
		abs, _ := filepath.Abs(path)
		if values, ok := inheritance.Map[abs]; ok {
			for _, value := range values {
				evaluate(value, inherit)
			}
		}
	}

	return nil
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

type Inheritance struct {
	Map  map[string]sort.StringSlice
	lock sync.Mutex
}

func (m *Inheritance) Contains(key, value string) bool {
	slice, contains := m.Map[key]
	if !contains {
		return false
	}

	idx := slice.Search(value)
	return idx < len(slice) && slice[idx] == value
}

func (m *Inheritance) Add(key, value string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.Contains(key, value) {
		return
	}

	m.Map[key] = append(m.Map[key], value)
	m.Map[key].Sort()
}
