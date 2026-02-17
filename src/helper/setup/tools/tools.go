package tools

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

// Global progress tracker for setup process
var setupProgress *fmtc.StepProgress

// InitProgress initializes the progress tracker with given steps
func InitProgress(steps []string) {
	setupProgress = fmtc.NewStepProgress(steps)
}

// SetProgressStep sets and displays the current step
func SetProgressStep(step int) {
	if setupProgress != nil {
		setupProgress.SetStep(step)
		setupProgress.Display()
	}
}

// CompleteProgress displays the completion message
func CompleteProgress() {
	if setupProgress != nil {
		setupProgress.Complete()
	}
}

// PlatformChoice represents a platform option in the interactive selector.
type PlatformChoice struct {
	Name        string // internal name, e.g. "magento2"
	DisplayName string // human-readable, e.g. "Magento 2"
}

func Platform(choices []PlatformChoice) string {
	displayNames := make([]string, len(choices))
	displayToName := make(map[string]string, len(choices))
	for i, c := range choices {
		displayNames[i] = c.DisplayName
		displayToName[c.DisplayName] = c.Name
	}

	defVersion := displayNames[0]

	fmt.Println("")
	SelectInteractive("Platform", displayNames, &defVersion)
	if name, ok := displayToName[defVersion]; ok {
		return name
	}
	return defVersion
}

func Php(defVersion *string) {
	availableVersions := []string{"Custom", "8.4", "8.3", "8.2", "8.1", "8.0", "7.4"}

	fmt.Println("")
	SelectInteractive("PHP Version", availableVersions, defVersion)
}

func Db(defVersion *string) {
	availableVersions := []string{"Custom", "11.4", "11.1", "10.6", "10.4", "10.3", "10.2"}

	fmt.Println("")
	SelectInteractive("Database (MariaDB)", availableVersions, defVersion)
}

func Composer(defVersion *string) {
	availableVersions := []string{"Custom", "1", "2"}

	fmt.Println("")
	SelectInteractive("Composer Version", availableVersions, defVersion)
}

func SearchEngine(defVersion *string) {
	availableVersions := []string{"OpenSearch", "Elasticsearch", "Do not use"}

	fmt.Println("")
	SelectInteractive("Search Engine", availableVersions, defVersion)
}

func Elastic(defVersion *string) {
	availableVersions := []string{"Custom", "8.17.6", "8.11.14", "8.4.3", "7.17.5", "7.16.3", "7.10.1"}

	fmt.Println("")
	SelectInteractive("Elasticsearch Version", availableVersions, defVersion)
}

func OpenSearch(defVersion *string) {
	availableVersions := []string{"Custom", "2.19.0", "2.12.0", "2.5.0", "1.2.0"}

	fmt.Println("")
	SelectInteractive("OpenSearch Version", availableVersions, defVersion)
}

func Redis(defVersion *string) {
	availableVersions := []string{"Custom", "8.0", "7.2", "7.0", "6.2", "6.0", "5.0"}

	fmt.Println("")
	SelectInteractive("Redis Version", availableVersions, defVersion)
}

func Valkey(defVersion *string) {
	availableVersions := []string{"Custom", "8.1.3"}

	fmt.Println("")
	SelectInteractive("Valkey Version", availableVersions, defVersion)
}

func RabbitMQ(defVersion *string) {
	availableVersions := []string{"Custom", "4.1", "3.13", "3.12", "3.9", "3.8", "3.7"}

	fmt.Println("")
	SelectInteractive("RabbitMQ Version", availableVersions, defVersion)
}

func Hosts(projectName string, defVersion *string, projectConf map[string]string) {
	host := strings.ToLower(projectName + projectConf["nginx/default_host_first_level"] + ":base")
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		var hostItems []string
		for _, hostItem := range hosts {
			hostItems = append(hostItems, hostItem["name"]+":"+hostItem["code"])
		}
		host = strings.Join(hostItems, " ")
	}

	*defVersion = host
	availableVersions := []string{"Custom", projectName + projectConf["nginx/default_host_first_level"] + ":base", "loc." + projectName + ".com:base"}

	fmt.Println("")
	fmt.Printf("%sFormat: domain.com:website_code%s\n", fmtc.Gray(), fmtc.ResetColor())
	SelectInteractive("Hosts Configuration", availableVersions, defVersion)
}

func NodeJs(defVersion *string) {
	availableVersions := []string{"Custom", "21.1.0", "20.19.0", "18.15.0", "16.20.0"}

	fmt.Println("")
	SelectInteractive("NodeJS Version", availableVersions, defVersion)
}

func Yarn(defVersion *string) {
	availableVersions := []string{"Custom", "3.6.4", "1.22.19"}

	fmt.Println("")
	SelectInteractive("Yarn Version", availableVersions, defVersion)
}

func Language() string {
	defVersion := "php"
	availableVersions := []string{"php", "nodejs", "python", "golang", "ruby", "none"}

	fmt.Println("")
	SelectInteractive("Primary Language", availableVersions, &defVersion)
	return defVersion
}

func PythonVersion(defVersion *string) {
	availableVersions := []string{"Custom", "3.13", "3.12", "3.11", "3.10"}

	fmt.Println("")
	SelectInteractive("Python Version", availableVersions, defVersion)
}

func GoVersion(defVersion *string) {
	availableVersions := []string{"Custom", "1.23", "1.22", "1.21", "1.20"}

	fmt.Println("")
	SelectInteractive("Go Version", availableVersions, defVersion)
}

func RubyVersion(defVersion *string) {
	availableVersions := []string{"Custom", "3.3", "3.2", "3.1", "3.0"}

	fmt.Println("")
	SelectInteractive("Ruby Version", availableVersions, defVersion)
}

func setTitleAndRecommended(title string, recommended *string) {
	// Title is now shown as part of the selector box
	fmt.Println("")
}

func PrepareVersions(availableVersions []string) {
	for index, ver := range availableVersions {
		if ver != "" {
			fmt.Println(strconv.Itoa(index) + ") " + ver)
		}
	}
}

// PrepareVersionsStyled displays versions in a styled selector box
func PrepareVersionsStyled(title string, availableVersions []string, recommended string) {
	var options []fmtc.SelectorOption
	recommendedKey := ""

	for index, ver := range availableVersions {
		if ver == "" {
			continue
		}
		key := strconv.Itoa(index)
		isRecommended := ver == recommended
		if isRecommended {
			recommendedKey = key
		}
		options = append(options, fmtc.SelectorOption{
			Key:         key,
			Value:       ver,
			Recommended: isRecommended,
		})
	}

	fmtc.Selector(title, options, recommendedKey)
}

// SelectInteractive runs an interactive selector and returns the selected value
func SelectInteractive(title string, availableVersions []string, defVersion *string) {
	// Find recommended index
	recommendedIdx := -1
	for i, ver := range availableVersions {
		if ver == *defVersion {
			recommendedIdx = i
			break
		}
	}

	selector := fmtc.NewInteractiveSelector(title, availableVersions, recommendedIdx)
	idx, value := selector.Run()

	// Handle "Custom" option
	if value == "Custom" {
		fmt.Print("Enter custom value: ")
		buf := bufio.NewReader(os.Stdin)
		sentence, err := buf.ReadBytes('\n')
		if err != nil {
			logger.Fatalln(err)
		}
		*defVersion = strings.TrimSpace(string(sentence))
		fmtc.SuccessLn("Your choice: " + *defVersion)
	} else {
		*defVersion = value
		// Result already printed by selector
		_ = idx
	}
}

func Invitation(ver *string) {
	if *ver != "" {
		fmt.Println("Enter the item number or press Enter to select the recommended item")
	} else {
		fmt.Println("Enter the item number")
	}

	fmt.Print("> ")
}

func WaiterAndProceed(defVersion *string, availableVersions []string) {
	selected, repoAndVersion := Waiter()
	setSelectedVersion(defVersion, availableVersions, selected, repoAndVersion)
}

func Waiter() (selected, repoAndVersion string) {
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		logger.Fatalln(err)
	}
	selected = strings.TrimSpace(string(sentence))
	if selected == "0" {
		fmt.Println("Enter the custom value")
		fmt.Print("> ")
		buf = bufio.NewReader(os.Stdin)
		sentence, err = buf.ReadBytes('\n')
		if err != nil {
			logger.Fatalln(err)
		}
		repoAndVersion = strings.TrimSpace(string(sentence))
	}

	return
}

func setSelectedVersion(defVersion *string, availableVersions []string, selected, repoAndVersion string) {
	selectedInt, err := strconv.Atoi(selected)
	if selected == "" && *defVersion != "" {
		fmtc.SuccessLn("Your choice: " + *defVersion)
	} else if selected == "0" {
		*defVersion = repoAndVersion
		fmtc.SuccessLn("Your choice: " + *defVersion)
	} else if selected != "" && err == nil && len(availableVersions) >= selectedInt {
		version := strings.Split(availableVersions[selectedInt], " ")
		*defVersion = version[0]
		fmtc.SuccessLn("Your choice: " + *defVersion)
	} else {
		fmtc.WarningLn("Choose one of the options offered")
		WaiterAndProceed(defVersion, availableVersions)
	}
}

func copyFile(pathFrom, pathTo string) {
	b, err := os.ReadFile(pathFrom)
	if err != nil {
		logger.Fatal(err)
	}
	pathToAsSlice := strings.Split(pathTo, "/")
	paths.MakeDirsByPath(strings.Join(pathToAsSlice[:len(pathToAsSlice)-1], "/"))
	err = os.WriteFile(pathTo, b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}
