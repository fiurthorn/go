package main

import (
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fiurthorn/go/lib"
	bh "github.com/timshannon/badgerhold/v4"

	// bh "github.com/timshannon/bolthold"
	"golang.org/x/crypto/sha3"
)

type Entry struct {
	ID   uint64 `badgerhold:"key" boltholdKey:"ID"`
	Hash string `badgerhold:"index" boltholdIndex:"Hash"`

	Files *lib.StringSet
}

var wg = sync.WaitGroup{}
var pool = make(chan string, 64)
var store *bh.Store

func main() {
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

	filepath.WalkDir(filepath.Join(home, "workspace"), walker)
	log.Println("walked through")
	close(pool)
	log.Println("waiting")
	wg.Wait()
	close(queue)
	<-ready
	d := time.Since(start)

	results := []Entry{}
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

	count, err := store.Count(Entry{}, &bh.Query{})
	if err != nil {
		log.Panic(err)
	}
	log.Println(len(results), "/", count)
	log.Println(insert, "/", visit)
	log.Println("time", d)
}

type FileHash struct {
	Hash string
	File string
}

var queue = make(chan FileHash, 20)
var ready = make(chan struct{})
var insert = 0

func inserter() {
	for entry := range queue {
		insert++
		var result Entry
		store.FindOne(&result, (*bh.Query)(bh.Where("Hash").Eq(entry.Hash).Index("Hash")))
		if result.ID > 0 || (result.Files != nil && result.Files.Len() > 0) {
			if has := result.Files.Has(entry.File); !has {
				result.Files.Add(entry.File)
				// log.Println("update", entry.File)
				store.Update(result.ID, result)
			}
			continue
		}

		data := Entry{
			Hash:  entry.Hash,
			Files: lib.NewStringSetWith(entry.File),
		}
		// log.Println("insert", entry.File)
		store.Insert(bh.NextSequence(), data)
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
		visit++
		pool <- path
	}

	return nil
}

func worker(i int, pool chan string, h hash.Hash) {
	for path := range pool {
		logHash(i, path, h)
	}
	wg.Done()
}

func logHash(i int, path string, h hash.Hash) string {
	hash := calcHash(path, h)
	log.Printf("[%2d]: %s: %s\r", i, hash, path)
	queue <- FileHash{Hash: hash, File: path}
	return hash
}

func calcHash(path string, h hash.Hash) string {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("skip dir '%s' with error: %v", path, err)
	}
	defer file.Close()

	h.Reset()
	io.Copy(h, file)
	return hex.EncodeToString(h.Sum(nil))
}
