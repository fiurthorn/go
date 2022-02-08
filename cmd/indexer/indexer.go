package main

import (
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fiurthorn/go/lib"
	bh "github.com/timshannon/badgerhold/v4"

	// bh "github.com/timshannon/bolthold"

	"golang.org/x/crypto/sha3"
)

type HashEntry struct {
	ID   uint64 `badgerhold:"key" boltholdKey:"ID"`
	Hash string `badgerhold:"index" boltholdIndex:"Hash"`

	Files *lib.StringSet
}

type FileEntry struct {
	ID   string `badgerhold:"key" boltholdKey:"ID"`
	Size int64  `badgerhold:"index" boltholdIndex:"Size"`
}

var wg = sync.WaitGroup{}
var pool = make(chan string, 64)
var store *bh.Store
var quit = false

func signalHandler(cancel chan os.Signal) {
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	for signal := range cancel {
		log.Print("terminate:", signal)
		quit = true
	}
}

func main() {
	go signalHandler(make(chan os.Signal))

	start := time.Now()
	go inserter()
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go worker(i, pool, sha3.New256())
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}
	dir := filepath.Join(home, "indexer")

	// //badger
	options := bh.DefaultOptions
	options.Dir = dir
	options.ValueDir = dir
	store, err = bh.Open(options)

	// bolt
	// store, err = bh.Open(filepath.Join(dir, "bold.db"), 0666, &bh.Options{})

	if err != nil {
		log.Panic(err)
	}
	defer store.Close()

	filepath.WalkDir(filepath.Join(home, "Downloads"), walker)
	log.Println("walked through")
	close(pool)
	log.Println("waiting")
	wg.Wait()
	close(queue)
	<-ready
	d := time.Since(start)

	if !quit {
		results := []HashEntry{}
		err = store.Find(&results, (*bh.Query)(bh.Where("Files").MatchFunc(func(ra *bh.RecordAccess) (bool, error) {
			if files, ok := ra.Field().(*lib.StringSet); ok {
				return files.Len() > 1, nil
			}
			return false, nil
		})))
		if err != nil {
			log.Panic(err)
		}

		for _, result := range results {
			log.Println(result)
		}

		count, err := store.Count(HashEntry{}, &bh.Query{})
		if err != nil {
			log.Panic(err)
		}
		count2, err := store.Count(FileEntry{}, &bh.Query{})
		if err != nil {
			log.Panic(err)
		}
		log.Println(len(results), "/", count, "/", count2)
	}
	log.Println(insert, "/", visit)
	log.Println("time", d)
}

type FileHash struct {
	Hash string
	File string
	Size int64
}

var queue = make(chan FileHash, 20)
var ready = make(chan struct{})
var insert = 0

func inserter() {
	for entry := range queue {
		insert++
		var result HashEntry
		store.FindOne(&result, (*bh.Query)(bh.Where("Hash").Eq(entry.Hash).Index("Hash")))
		if result.ID > 0 || (result.Files != nil && result.Files.Len() > 0) {
			if has := result.Files.Has(entry.File); !has {
				result.Files.Add(entry.File)
				// log.Println("update", entry.File)
				store.Update(result.ID, result)
				store.Insert(entry.File, FileEntry{Size: entry.Size})
			}
			continue
		}

		data := HashEntry{
			Hash:  entry.Hash,
			Files: lib.NewStringSetWith(entry.File),
		}
		// log.Println("insert", entry.File)
		store.Insert(bh.NextSequence(), data)
		store.Insert(entry.File, FileEntry{Size: entry.Size})
	}
	ready <- struct{}{}
}

var visit = 0

func walker(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return fmt.Errorf("skip dir '%s' with error: %w", path, err)
	}

	if d.IsDir() && filepath.Base(path) == ".git" {
		return fs.SkipDir
	}

	if d.Type().IsRegular() {
		if !quit {
			visit++
			pool <- path
		}
	}

	return nil
}

func worker(i int, pool chan string, h hash.Hash) {
	for path := range pool {
		if !quit {
			logHash(i, path, h)
		}
	}
	wg.Done()
}

func logHash(i int, path string, h hash.Hash) string {
	start := time.Now()
	hash, size := calcHash(path, h)
	log.Printf("[%2d]: [%10s] %s: %s\r", i, time.Since(start), hash, path)
	queue <- FileHash{Hash: hash, File: path, Size: size}
	return hash
}

func calcHash(path string, h hash.Hash) (string, int64) {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("skip dir '%s' with error: %v", path, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Printf("skip dir '%s' with error: %v", path, err)
	}
	size := stat.Size()

	h.Reset()
	io.Copy(h, file)
	return hex.EncodeToString(h.Sum(nil)), size
}
