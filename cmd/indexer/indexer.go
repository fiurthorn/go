package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fiurthorn/go/lib"
	bh "github.com/timshannon/badgerhold/v4"
	"golang.org/x/crypto/sha3"
)

type Entry struct {
	ID   uint64 `badgerhold:"key"`
	Hash string `badgerhold:"index"`

	Files *lib.StringSet
}

var wg = sync.WaitGroup{}
var pool = make(chan string, 64)
var store *bh.Store

func main() {
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go worker(i, pool)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}
	dir := filepath.Join(home, "indexer")

	options := bh.DefaultOptions
	options.Dir = dir
	options.ValueDir = dir

	store, err = bh.Open(options)
	if err != nil {
		log.Panic(err)
	}
	defer store.Close()

	filepath.WalkDir("/home/fiurthorn/workspace/xxo", walker)
	log.Println("walked through")
	close(pool)
	log.Println("waiting")
	wg.Wait()

	results := []Entry{}
	store.Find(&results, (*bh.Query)(bh.Where("Files").MatchFunc(func(ra *bh.RecordAccess) (bool, error) {
		if files, ok := ra.Field().(*lib.StringSet); ok {
			return files.Len() > 1, nil
		}
		return false, nil
	})))

	for _, result := range results {
		log.Println(result)
	}
	log.Println(len(results))
}

func walker(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return fmt.Errorf("skip dir '%s' with error: %w", path, err)
	}

	if d.IsDir() && filepath.Base(path) == ".git" {
		return fs.SkipDir
	}

	if d.Type().IsRegular() {
		pool <- path
	}

	return nil
}

func worker(i int, pool chan string) {
	for path := range pool {
		logHash(i, path)
	}
	wg.Done()
}

func logHash(i int, path string) string {
	hash := calcHash(path)
	log.Printf("[%2d]: %s: %s\r", i, hash, path)

	var result Entry
	store.FindOne(&result, (*bh.Query)(bh.Where("Hash").Eq(hash)))
	if result.ID > 0 || (result.Files != nil && result.Files.Len() > 0) {
		if has := result.Files.Has(path); !has {
			result.Files.Add(path)
			store.Update(result.ID, result)
		}
		return hash
	}

	data := Entry{
		Hash:  hash,
		Files: lib.NewStringSetWith(path),
	}
	store.Insert(bh.NextSequence(), data)

	return hash
}

func calcHash(path string) string {
	hash := sha3.New256()
	file, err := os.Open(path)
	if err != nil {
		log.Printf("skip dir '%s' with error: %v", path, err)
	}
	defer file.Close()

	io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil))
}
