package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/crypto/sha3"
)

func main() {
	filepath.WalkDir("c:/Users/s.weinmann/workspace", walker)
}

func walker(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return fmt.Errorf("skip dir '%s' with error: %w", path, err)
	}

	if d.IsDir() && filepath.Base(path) == ".git" {
		return fs.SkipDir
	}

	if d.Type().IsRegular() {
		hash := sha3.New256()
		file, err := os.Open(path)
		if err != nil {
			log.Printf("skip dir '%s' with error: %v", path, err)
		}
		defer file.Close()

		io.Copy(hash, file)
		bytes := hash.Sum(nil)
		log.Printf("%s: %s\r", hex.EncodeToString(bytes), path)
	}

	return nil
}
