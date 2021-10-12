package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type Ref struct {
	Ref    string `json:"ref"`
	NodeId string `json:"node_id"`
	Url    string `json:"url"`
	Object struct {
		Sha  string `json:"sha"`
		Type string `json:"type"`
		Url  string `json:"url"`
	} `json:"object"`
}

func githubUserProject() (string, string) {
	return "golang", "go"
}

func githubApiUrl() string {
	user, project := githubUserProject()
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs/tags", user, project)
}

func versionExtractPrefix() string {
	return "go"
}

func versionExtractRegex() *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`^refs/tags/(%s\d+(\.\d+(\.\d+)?)?)$`, versionExtractPrefix()))
}

func convert(output []byte, err error) (string, error) {
	return string(output), err
}

func currentGoVersion() string {
	executable, err := exec.LookPath("go")
	if err != nil {
		return "0"
	}
	proc := exec.Cmd{Path: executable, Args: []string{executable, "version"}}
	outputString, err := convert(proc.CombinedOutput())
	if err != nil {
		log.Panic(err.Error())
	}
	outputString = strings.Split(outputString, " ")[2]
	return outputString

}

func currentVersion() string {
	return currentGoVersion()
}

func downloadUrl(version string) (url string, filename string) {
	filename = fmt.Sprintf(`%s.%s-%s.%s`, version, runtime.GOOS, runtime.GOARCH, goLangExtention())
	log.Println("select:", filename)
	url = fmt.Sprintf(`https://golang.org/dl/%s`, filename)
	return
}

func goLangExtention() string {
	switch runtime.GOOS {
	case "darwin":
		return "pkg"
	case "linux":
		return "tar.gz"
	case "windows":
		return "msi"
	default:
		log.Fatalf("%s not supported!", runtime.GOOS)
		return ""
	}
}

func main() {
	refs := &[]Ref{}
	client := &http.Client{}

	log.Println("get github tags")
	req, err := http.NewRequest("GET", githubApiUrl(), nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Accepted", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	json.NewDecoder(resp.Body).Decode(refs)

	log.Println("filter github tags")
	validRef := versionExtractRegex()
	extract := make([]string, 0, len(*refs))
	for _, ref := range *refs {
		if validRef.MatchString(ref.Ref) {
			content := validRef.FindAllStringSubmatch(ref.Ref, -1)
			extract = append(extract, content[0][1])
		}
	}

	extractVersion := extract[len(extract)-1]
	log.Println("filtered:", extractVersion)

	log.Printf("extractVersion:%s, currentVersion:%s", extractVersion, currentVersion())
	if currentVersion() == extractVersion {
		log.Printf("latest version already installed (%s==%s)", extractVersion, currentVersion())
	} else {
		url, filename := downloadUrl(extractVersion)
		log.Print(url)

		res, err := client.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		outFile, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, res.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("last version downloaded: (%s!=%s)", extractVersion, runtime.Version())

		fullpath, err := filepath.Abs(filename)
		if err != nil {
			log.Fatal(err)
		}

		executable, err := lookupExecutable()
		if err != nil {
			log.Fatal(err)
		}

		dirname := filepath.Dir(executable)
		if strings.HasSuffix(dirname, "bin") {
			dirname = filepath.Dir(executable)
		}
		log.Printf("install location: %s => %s", fullpath, dirname)
	}
}

func lookupExecutable() (string, error) {
	return exec.LookPath("go")
}
