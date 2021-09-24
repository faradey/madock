package commands

import (
	"bufio"
	"fmt"
	"github.com/faradey/madock/src/functions"
	"github.com/faradey/madock/src/versions"
	"log"
	"os"
	"strconv"
	"strings"
)

func Setup() {
	fmt.Println("Start set up environment")
	toolsDefVersions := versions.GetVersions()
	buf := bufio.NewReader(os.Stdin)
	if toolsDefVersions.Php != "" {
		fmt.Println("Recommended PHP version: " + toolsDefVersions.Php)
	}

	phpVersionList := []string{"8.1", "8.0", "7.4", "7.3", "7.2", "7.1", "7.0"}

	for index, ver := range phpVersionList {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	if toolsDefVersions.Php != "" {
		fmt.Println("Enter the item number or press Enter to select the recommended version")
	} else {
		fmt.Println("Enter the item number")
	}
	fmt.Print("> ")
	sentence, err := buf.ReadBytes('\n')
	selected := strings.TrimSpace(string(sentence))
	if err != nil {
		log.Fatalln(err)
	} else {
		if selected == "" && toolsDefVersions.Php != "" {
			fmt.Println("Your choice: " + toolsDefVersions.Php)
		} else if selected != "" && functions.IsContain(phpVersionList, selected) {
			toolsDefVersions.Php = selected
			fmt.Println("Your choice: " + selected)
		} else {
			log.Fatalln("This PHP version is not supported")
		}
	}

	buf = bufio.NewReader(os.Stdin)
	if toolsDefVersions.Php != "" {
		fmt.Println("Recommended MariaDB version: " + toolsDefVersions.Db)
	}

	dbVersionList := []string{"10.4", "10.3", "10.2", "10.1", "10.0"}

	for index, ver := range dbVersionList {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	if toolsDefVersions.Db != "" {
		fmt.Println("Enter the item number or press Enter to select the recommended version")
	} else {
		fmt.Println("Enter the item number")
	}
	fmt.Print("> ")
	sentence, err = buf.ReadBytes('\n')
	selected = strings.TrimSpace(string(sentence))
	if err != nil {
		log.Fatalln(err)
	} else {
		if selected == "" && toolsDefVersions.Db != "" {
			fmt.Println("Your choice: " + toolsDefVersions.Db)
		} else if selected != "" && functions.IsContain(dbVersionList, selected) {
			toolsDefVersions.Db = selected
			fmt.Println("Your choice: " + selected)
		} else {
			log.Fatalln("This MariaDB version is not supported")
		}
	}

	fmt.Println("Finish set up environment")
}

func IsNotDefine() {
	fmt.Println("The command is not defined. Run 'madock help' to invoke help")
}
