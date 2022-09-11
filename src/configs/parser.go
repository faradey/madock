package configs

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func ParseFile(path string) (conf map[string]string) {
	conf = make(map[string]string)
	lines := getLines(path)

	for _, line := range lines {
		opt := strings.Split(strings.TrimSpace(line), "=")
		if len(opt) > 1 {
			conf[opt[0]] = opt[1]
		} else if len(opt) > 0 {
			conf[opt[0]] = ""
		}
	}

	return conf
}

func getLines(path string) []string {
	var rows []string
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) > 0 && strings.TrimSpace(line)[:1] != "#" {
			rows = append(rows, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return rows
}

func GetAllLines(path string) []string {
	var rows []string
	file, err := os.Open(path)
	if err != nil {
		return rows
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			rows = append(rows, "")
		} else if strings.TrimSpace(line)[:1] != "#" {
			rows = append(rows, line)
		} else {
			rows = append(rows, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return rows
}
