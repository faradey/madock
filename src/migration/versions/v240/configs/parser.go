package configs

import (
	"bufio"
	"github.com/faradey/madock/src/helper/logger"
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
		logger.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
			rows = append(rows, line)
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Fatal(err)
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
		rows = append(rows, line)
	}

	if err = scanner.Err(); err != nil {
		logger.Fatal(err)
	}

	return rows
}
