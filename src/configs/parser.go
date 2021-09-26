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
		opt := strings.Split(line, "=")
		if len(opt) > 1 {
			conf[opt[0]] = opt[1]
		} else {
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
		if strings.TrimSpace(line[:1]) != "#" {
			rl := len(rows)
			if rl > 0 && rows[rl-1][len(rows[rl-1])-1:] == "\\" {
				rows[rl-1] = rows[rl-1][:len(rows[rl-1])-1] + line
			} else {
				rows = append(rows, line)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return rows
}
