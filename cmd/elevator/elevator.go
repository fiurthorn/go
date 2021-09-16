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

func githubUserProject(golang bool) (string, string) {
	if golang {
		return "golang", "go"
	}
	return "dart-lang", "sdk"
}

func githubApiUrl(golang bool) string {
	user, project := githubUserProject(golang)
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs/tags", user, project)
}

func versionExtractPrefix(golang bool) string {
	if golang {
		return "go"
	}
	return ""
}

func versionExtractRegex(golang bool) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`^refs/tags/(%s\d+(\.\d+(\.\d+)?)?)$`, versionExtractPrefix(golang)))
}

func convert(output []byte, err error) (string, error) {
	return string(output), err
}

func currentDartVersion() string {
	executable, err := exec.LookPath("dart")
	if err != nil {
		return "0"
	}
	proc := exec.Cmd{Path: executable, Args: []string{executable, "--version"}}
	outputString, err := convert(proc.CombinedOutput())
	if err != nil {
		log.Panic(err.Error())
	}
	outputString = strings.TrimPrefix(string(outputString), "Dart SDK version: ")
	outputString = strings.Split(outputString, " ")[0]
	return outputString
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

func currentVersion(golang bool) string {
	if golang {
		return currentGoVersion()
	}

	return currentDartVersion()
}

func downloadUrl(golang bool, version string) (url string, filename string) {
	if golang {
		filename = fmt.Sprintf(`%s.%s-%s.%s`, version, runtime.GOOS, runtime.GOARCH, goLangExtention())
		log.Println("select:", filename)
		url = fmt.Sprintf(`https://golang.org/dl/%s`, filename)
		return
	}

	filename = fmt.Sprintf("dartsdk-%s-%s-%s.zip", dartOs(), dartArch(), version)
	url = fmt.Sprintf("https://storage.googleapis.com/dart-archive/channels/stable/release/%s/sdk/dartsdk-%s-%s-release.zip", version, dartOs(), dartArch())
	return
}

func dartOs() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	default:
		return runtime.GOOS
	}
}

func dartArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	case "386":
		return "ia32"
	default:
		return runtime.GOARCH
	}
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
	args := os.Args[1:]
	golang := true
	if len(args) > 0 && args[0] == "dart" {
		golang = false
	}
	refs := &[]Ref{}
	client := &http.Client{}

	log.Println("get github tags")
	req, err := http.NewRequest("GET", githubApiUrl(golang), nil)
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
	validRef := versionExtractRegex(golang)
	extract := make([]string, 0, len(*refs))
	for _, ref := range *refs {
		if validRef.MatchString(ref.Ref) {
			content := validRef.FindAllStringSubmatch(ref.Ref, -1)
			extract = append(extract, content[0][1])
		}
	}

	extractVersion := extract[len(extract)-1]
	log.Println("filtered:", extractVersion)

	log.Printf("extractVersion:%s, currentVersion:%s", extractVersion, currentVersion(golang))
	if currentVersion(golang) == extractVersion {
		log.Printf("latest version already installed (%s==%s)", extractVersion, currentVersion(golang))
	} else {
		url, filename := downloadUrl(golang, extractVersion)
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

		if strings.HasSuffix(filename, ".msi") {
			installMsi(fullpath)
		} else {
			executable, err := lookupExecutable(golang)
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
}

func lookupExecutable(golang bool) (string, error) {
	if golang {
		return exec.LookPath("go")
	}

	return exec.LookPath("dart")
}

func installMsi(fullpath string) {
	msiexec, err := exec.LookPath("msiexec")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s %s", msiexec, fullpath)
	_, err = exec.Command(msiexec, "/package", fullpath).CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

}
