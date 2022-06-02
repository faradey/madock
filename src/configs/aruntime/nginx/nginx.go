package nginx

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/functions"
	"github.com/faradey/madock/src/paths"
)

func MakeConf() {
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/docker/nginx")
	setPorts()
	makeProxy()
	makeDockerfile()
	makeDockerCompose()
}

func setPorts() {
	projectsAruntime := paths.GetDirs(paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects"))
	projects := paths.GetDirs(paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects"))
	if len(projectsAruntime) > len(projects) {
		projects = projectsAruntime
	}
	portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
	portsConfig := make(map[string]string)
	if _, err := os.Stat(portsFile); os.IsNotExist(err) {
		lines := ""
		for port, line := range projects {
			lines += line + "=" + strconv.Itoa(port+1) + "\n"
		}
		err = ioutil.WriteFile(portsFile, []byte(lines), 0755)
	}

	portsConfig = configs.ParseFile(portsFile)

	if _, ok := portsConfig[paths.GetRunDirName()]; !ok {
		f, err := os.OpenFile(portsFile,
			os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Println(err)
		}
		defer f.Close()
		maxPort := getMaxPort(portsConfig)
		if _, err := f.WriteString(paths.GetRunDirName() + "=" + strconv.Itoa(maxPort+1) + "\n"); err != nil {
			log.Println(err)
		}
	}
}

func makeProxy() {
	portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
	portsConfig := configs.ParseFile(portsFile)
	/* Create nginx default configuration for Magento2 */
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/conf/default-proxy.conf"
	allFileData := "worker_processes 2;\nworker_priority -10;\nworker_rlimit_nofile 200000;\nevents {\n    worker_connections 4096;\nuse epoll;\n}\nhttp {\n"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	projectsNames := paths.GetDirs(paths.GetExecDirPath() + "/aruntime/projects")
	for _, name := range projectsNames {
		if _, err := os.Stat(paths.GetExecDirPath() + "/aruntime/projects/" + name + "/stopped"); os.IsNotExist(err) {
			port, err := strconv.Atoi(portsConfig[name])
			if err != nil {
				log.Fatal(err)
			}
			portRanged := (port - 1) * 20
			strReplaced := strings.Replace(str, "{{{NGINX_PORT}}}", strconv.Itoa(17000+portRanged), -1)
			for i := 1; i < 20; i++ {
				strReplaced = strings.Replace(strReplaced, "{{{NGINX_PORT+"+strconv.Itoa(i)+"}}}", strconv.Itoa(17000+portRanged+i), -1)
			}
			hostName := "loc." + name + ".com"
			projectConf := configs.GetProjectConfig(name)
			if val, ok := projectConf["HOSTS"]; ok {
				var onlyHosts []string
				hosts := strings.Split(val, " ")
				if len(hosts) > 0 {
					for _, hostAndStore := range hosts {
						onlyHosts = append(onlyHosts, strings.Split(hostAndStore, ":")[0])
					}
					hostName = strings.Join(onlyHosts, "\n")
				}
			}

			strReplaced = strings.Replace(strReplaced, "{{{HOST_NAMES}}}", hostName, -1)
			allFileData += strReplaced
		}
	}
	allFileData += "\n}"
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/ctx") + "/proxy.conf"
	err = ioutil.WriteFile(nginxFile, []byte(allFileData), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx default configuration for Magento2 */
}

func makeDockerfile() {
	/* Create nginx Dockerfile configuration */
	ctxPath := paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/proxy.Dockerfile"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	//str := string(b)
	projectsNames := paths.GetDirs(paths.GetExecDirPath() + "/aruntime/projects")
	var commands []string
	var i int = 0
	for _, name := range projectsNames {
		projectConf := configs.GetProjectConfig(name)
		if val, ok := projectConf["HOSTS"]; ok {
			var onlyHost string
			hosts := strings.Split(val, " ")
			if len(hosts) > 0 {
				for _, hostAndStore := range hosts {
					onlyHost = strings.Split(hostAndStore, ":")[0]
					commands = append(commands, "DNS."+strconv.Itoa(i)+" = "+onlyHost)
					i++
				}
			}
		}
	}
	/*commandsString := "RUN " + strings.Join(commands, " && ")
	str = strings.Replace(str, "{{{SSL_CREATE_BY_HOST_NAMES}}}", commandsString, -1)
	*/
	err = ioutil.WriteFile(ctxPath+"/Dockerfile", b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	err = ioutil.WriteFile(ctxPath+"/rsapassword.txt", []byte(functions.GeneratePassword(15, 3, 4, 3)), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	extFileContent := "authorityKeyIdentifier=keyid,issuer\n" +
		"basicConstraints=CA:FALSE\n" +
		"keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment\n" +
		"subjectAltName = @alt_names\n" +
		"\n" +
		"[alt_names]\n" +
		"" + strings.Join(commands, "\n")

	err = ioutil.WriteFile(ctxPath+"/madock.ca.ext", []byte(extFileContent), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:4096", "-keyout", ctxPath+"/madockCA.key", "-out", ctxPath+"/madockCA.pem", "-sha256", "-days", "1800", "-nodes", "-subj", "/CN=madock")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Enter your password for adding ssl certificate to your system.")

	cmd = exec.Command("sudo", "security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", ctxPath+"/madockCA.pem")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	/* END Create nginx Dockerfile configuration */
}

func makeDockerCompose() {
	/* Copy nginx docker-compose configuration */
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/docker-compose-proxy.yml"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/docker-compose.yml", b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}

func getMaxPort(conf map[string]string) int {
	max := 0
	portInt := 0
	var err error
	for _, port := range conf {
		portInt, err = strconv.Atoi(port)
		if err != nil {
			log.Fatal(err)
		}
		if max < portInt {
			max = portInt
		}
	}

	return max
}
