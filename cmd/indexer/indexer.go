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

	"golang.org/x/crypto/sha3"
)

var wg = sync.WaitGroup{}
var pool = make(chan string, 64)

func main() {
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go worker(i, pool)
	}
	filepath.WalkDir("c:/Users/s.weinmann/workspace", walker)
	log.Println("walked through")
	close(pool)
	log.Println("waiting")
	wg.Wait()
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
