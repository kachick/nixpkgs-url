package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
)

const version string = "0.1.0"

var revision string

func main() {
	versionFlag := flag.Bool("version", false, "print the version of this program")
	currentFlag := flag.Bool("current", false, "print current nixpath without bumping")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("%s\n", version+"("+revision+")")
		return
	}

	path := "default.nix"
	isNixFileExist := true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = "shell.nix"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			isNixFileExist = false
		}
	}

	if isNixFileExist {
		if *currentFlag {
			current, err := getCurrentVersion(path)
			if err != nil {
				log.Fatalf("Extracting the version has been failed: %s", err.Error())
			}
			fmt.Println(current)
			return
		}
		err := bump(path)
		if err != nil {
			log.Fatalf("Bumping the version has been failed: %s", err.Error())
		}
	} else {
		log.Fatalln("Both default.nix and shell.nix are not found")
	}
}

type Commit struct {
	Sha string `json:"sha"`
}

type Response struct {
	Commit Commit `json:"commit"`
}

var re = regexp.MustCompile(`(?s)(import\s+\(fetchTarball\s+"https://github.com/NixOS/nixpkgs/archive/)([^"]+?)(\.tar\.gz"\))`)

func bump(path string) error {
	origin, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/NixOS/nixpkgs/branches/master", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	client := new(http.Client)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	jsonRes := &Response{}
	if json.Unmarshal(body, jsonRes) != nil {
		return err
	}
	replaced := re.ReplaceAll(origin, []byte("${1}"+jsonRes.Commit.Sha+"${3}"))
	if bytes.Equal(origin, replaced) {
		return nil
	}

	return os.WriteFile(path, replaced, os.ModePerm)
}

func getCurrentVersion(path string) (string, error) {
	origin, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	matches := re.FindStringSubmatch(string(origin))
	return matches[2], nil
}