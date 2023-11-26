package nginx

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/faradey/madock/src/configs/aruntime/project"
	"github.com/faradey/madock/src/helper"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/faradey/madock/src/configs"
)

func MakeConf() {
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/docker/nginx")
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
	if _, err := os.Stat(portsFile); os.IsNotExist(err) {
		lines := ""
		for port, line := range projects {
			lines += line + "=" + strconv.Itoa(port+1) + "\n"
		}
		_ = os.WriteFile(portsFile, []byte(lines), 0664)
	}

	portsConfig := configs.ParseFile(portsFile)
	lines := ""
	for projectName, port := range portsConfig {
		if _, err := os.Stat(paths.GetExecDirPath() + "/projects/" + projectName); !os.IsNotExist(err) {
			lines += projectName + "=" + port + "\n"
		}
	}

	if lines != "" {
		_ = os.WriteFile(portsFile, []byte(lines), 0664)
	}

	if _, ok := portsConfig[configs.GetProjectName()]; !ok {
		f, err := os.OpenFile(portsFile,
			os.O_APPEND|os.O_WRONLY, 0664)
		if err != nil {
			log.Println(err)
		}
		defer f.Close()
		maxPort := getMaxPort(portsConfig)
		if _, err := f.WriteString(configs.GetProjectName() + "=" + strconv.Itoa(maxPort+1) + "\n"); err != nil {
			log.Println(err)
		}
	}
}

func makeProxy() {
	portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
	portsConfig := configs.ParseFile(portsFile)
	generalConfig := configs.GetGeneralConfig()
	/* Create nginx default configuration for Magento2 */
	nginxDefFile := ""
	str := ""
	allFileData := "worker_processes 2;\nworker_priority -10;\nworker_rlimit_nofile 200000;\nevents {\n    worker_connections 4096;\nuse epoll;\n}\nhttp {\n"

	var onlyHostsGlobal []string
	projectsNames := paths.GetDirs(paths.GetExecDirPath() + "/aruntime/projects")
	if !helper.IsContain(projectsNames, configs.GetProjectName()) {
		projectsNames = append(projectsNames, configs.GetProjectName())
	}
	for _, name := range projectsNames {
		if _, err := os.Stat(paths.GetExecDirPath() + "/projects/" + name + "/env.txt"); !os.IsNotExist(err) {
			if _, err = os.Stat(paths.GetExecDirPath() + "/aruntime/projects/" + name + "/stopped"); os.IsNotExist(err) {
				nginxDefFile = project.GetDockerConfigFile(name, "/nginx/conf/default-proxy.conf", "general")
				b, err := os.ReadFile(nginxDefFile)
				if err != nil {
					log.Fatal(err)
				}

				str = string(b)
				port, err := strconv.Atoi(portsConfig[name])
				if err != nil {
					fmt.Println("Project name is " + name)
					log.Fatal(err)
				}
				portRanged := (port - 1) * 20
				strReplaced := strings.Replace(str, "{{{NGINX_PORT}}}", strconv.Itoa(17000+portRanged), -1)
				strReplaced = strings.Replace(strReplaced, "{{{NGINX_UNSECURE_PORT}}}", generalConfig["NGINX_UNSECURE_PORT"], -1)
				strReplaced = strings.Replace(strReplaced, "{{{NGINX_SECURE_PORT}}}", generalConfig["NGINX_SECURE_PORT"], -1)
				for i := 1; i < 20; i++ {
					strReplaced = strings.Replace(strReplaced, "{{{NGINX_PORT+"+strconv.Itoa(i)+"}}}", strconv.Itoa(17000+portRanged+i), -1)
				}
				strReplaced = configs.ReplaceConfigValue(strReplaced)
				hostName := "loc." + name + ".com"
				projectConf := configs.GetProjectConfig(name)
				if val, ok := projectConf["HOSTS"]; ok {
					var onlyHosts []string
					hosts := strings.Split(val, " ")
					if len(hosts) > 0 {
						domain := ""
						for _, hostAndStore := range hosts {
							domain = strings.Split(hostAndStore, ":")[0]
							if helper.IsContain(onlyHostsGlobal, domain) {
								log.Fatalln("Error. Duplicate domain " + domain)
							}
							onlyHosts = append(onlyHosts, domain)
							onlyHostsGlobal = append(onlyHostsGlobal, domain)
						}
						hostName = strings.Join(onlyHosts, "\n")
					}
				}

				strReplaced = strings.Replace(strReplaced, "{{{HOST_NAMES}}}", hostName, -1)
				allFileData += "\n" + strReplaced
			}
		}
	}

	allFileData += "\nserver {\n    listen       " + generalConfig["NGINX_UNSECURE_PORT"] + "  default_server;\n listen " + generalConfig["NGINX_SECURE_PORT"] + " default_server ssl;\n    server_name  _;\n    return       444;\n ssl_certificate /sslcert/fullchain.crt;\n        ssl_certificate_key /sslcert/madock.local.key;\n        include /sslcert/options-ssl-nginx.conf; \n}\n"
	allFileData += "\n}"
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/ctx") + "/proxy.conf"
	err := os.WriteFile(nginxFile, []byte(allFileData), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx default configuration for Magento2 */
}

func makeDockerfile() {
	/* Create nginx Dockerfile configuration */
	ctxPath := paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/general/nginx/proxy.Dockerfile"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs.ReplaceConfigValue(str)

	err = os.WriteFile(ctxPath+"/Dockerfile", []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}

func makeDockerCompose() {
	/* Copy nginx docker-compose configuration */
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/general/nginx/docker-compose-proxy.yml"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	str = configs.ReplaceConfigValue(str)

	err = os.WriteFile(paths.GetExecDirPath()+"/aruntime/docker-compose.yml", []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}

func getMaxPort(conf map[string]string) int {
	max := 0
	portInt := 0
	var err error
	var ports []int
	for _, port := range conf {
		portInt, err = strconv.Atoi(port)
		if err != nil {
			log.Fatal(err)
		}
		ports = append(ports, portInt)
	}

	for i := 1; i < 1000; i++ {
		if !helper.IsContainInt(ports, i) {
			return i - 1
		}
	}

	return max
}

func GenerateSslCert(ctxPath string, force bool) {
	generalConfig := configs.GetGeneralConfig()
	if val, ok := generalConfig["SSL"]; force || (ok && val == "true") {
		projectsNames := paths.GetDirs(paths.GetExecDirPath() + "/aruntime/projects")
		var commands []string
		i := 0
		for _, name := range projectsNames {
			if _, err := os.Stat(paths.GetExecDirPath() + "/projects/" + name + "/env.txt"); os.IsNotExist(err) {
				continue
			}

			projectConf := configs.GetProjectConfig(name)
			val := ""
			if val, ok = projectConf["HOSTS"]; !ok {
				continue
			}

			var onlyHost string
			hosts := strings.Split(val, " ")
			if len(hosts) == 0 {
				continue
			}

			for _, hostAndStore := range hosts {
				onlyHost = strings.Split(hostAndStore, ":")[0]
				commands = append(commands, "DNS."+strconv.Itoa(i+2)+" = "+onlyHost)
				i++
			}
		}

		extFileContent := "authorityKeyIdentifier=keyid,issuer\n" +
			"basicConstraints=CA:FALSE\n" +
			"keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment\n" +
			"subjectAltName = @alt_names\n" +
			"\n" +
			"[alt_names]\n" +
			"DNS.1 = madocklocalkey\n" +
			strings.Join(commands, "\n")

		err := os.WriteFile(ctxPath+"/madock.ca.ext", []byte(extFileContent), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}

		sslConfigFileContent := "ssl_session_cache shared:le_nginx_SSL:1m;\n" +
			"ssl_session_timeout 1440m;\n" +
			"\n" +
			"ssl_protocols TLSv1 TLSv1.1 TLSv1.2;\n" +
			"ssl_prefer_server_ciphers on;\n" +
			"\n" +
			"ssl_ciphers \"ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES256-SHA384:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA:ECDHE-RSA-AES256-SHA:DHE-RSA-AES128-SHA256:DHE-RSA-AES128-SHA:DHE-RSA-AES256-SHA256:DHE-RSA-AES256-SHA:ECDHE-ECDSA-DES-CBC3-SHA:ECDHE-RSA-DES-CBC3-SHA:EDH-RSA-DES-CBC3-SHA:AES128-GCM-SHA256:AES256-GCM-SHA384:AES128-SHA256:AES256-SHA256:AES128-SHA:AES256-SHA:DES-CBC3-SHA:!DSS\";"

		err = os.WriteFile(ctxPath+"/options-ssl-nginx.conf", []byte(sslConfigFileContent), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}

		doGenerateSsl := false
		if _, err := os.Stat(ctxPath + "/madockCA.pem"); os.IsNotExist(err) {
			doGenerateSsl = true
		} else {
			certificateCreatedTime, err := os.Stat(ctxPath + "/madockCA.pem")
			if err == nil && certificateCreatedTime.ModTime().Unix() < time.Now().Unix()-363*86400 {
				doGenerateSsl = true
			}
		}

		if doGenerateSsl || force {
			cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:4096", "-keyout", ctxPath+"/madockCA.key", "-out", ctxPath+"/madockCA.pem", "-sha256", "-days", "365", "-nodes", "-subj", "/CN=madock")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Enter your password for adding an SSL certificate to your system.")

			if runtime.GOOS == "darwin" {
				cmd = exec.Command("sudo", "security", "delete-certificate", "-c", "madock")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				_ = cmd.Run()

				cmd = exec.Command("sudo", "security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", ctxPath+"/madockCA.pem")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					log.Fatal(err)
				}
			} else if runtime.GOOS == "linux" {
				cmd = exec.Command("sudo", "cp", ctxPath+"/madockCA.pem", "/usr/local/share/ca-certificates/madockCA.crt")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					log.Fatal(err)
				}

				cmd = exec.Command("sudo", "chmod", "644", "/usr/local/share/ca-certificates/madockCA.crt")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					log.Fatal(err)
				}

				cmd = exec.Command("certutil", "-H")
				var outb, errb bytes.Buffer
				cmd.Stdout = &outb
				cmd.Stderr = &errb
				err = cmd.Run()
				selected := "y"
				if err != nil && errb.String() == "" {
					fmt.Println("You need to install \"certutil\" to proceed with the certificate installation. Continue installation? y - continue. n - cancel certificate generation and continue without ssl.")
					fmt.Print("> ")
					buf := bufio.NewReader(os.Stdin)
					sentence, err := buf.ReadBytes('\n')
					if err != nil {
						log.Fatalln(err)
					}
					selected = strings.TrimSpace(string(sentence))
					if selected == "y" {
						cmd = exec.Command("sudo", "apt", "install", "-y", "libnss3-tools")
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						err = cmd.Run()
						if err != nil {
							log.Fatal(err)
						}
					}
				}

				if selected == "y" {
					usr, _ := user.Current()
					if _, err := os.Stat(usr.HomeDir + "/.pki/nssdb"); os.IsNotExist(err) {
						paths.MakeDirsByPath(usr.HomeDir + "/.pki/nssdb")
						err = os.WriteFile(ctxPath+"/certutil_db_passwd.txt", []byte(""), 0755)
						if err != nil {
							cmd = exec.Command("certutil", "-d", usr.HomeDir+"/.pki/nssdb", "-N", ctxPath+"/certutil_db_passwd.txt")
							cmd.Stdout = os.Stdout
							cmd.Stderr = os.Stderr
							_ = cmd.Run()
						}
					}
					cmd = exec.Command("certutil", "-d", "sql:"+usr.HomeDir+"/.pki/nssdb", "-A", "-t", "C,,", "-n", "madocklocalkey", "-i", ctxPath+"/madockCA.pem")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err = cmd.Run()
					if err != nil {
						log.Fatal(err)
					}
				}

				cmd = exec.Command("sudo", "update-ca-certificates", "-f")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					log.Fatal(err)
				}
			}
		}

		cmd := exec.Command("openssl", "req", "-newkey", "rsa:4096", "-keyout", ctxPath+"/madock.local.key", "-out", ctxPath+"/madock.local.csr", "-nodes", "-subj", "/CN=madocklocalkey")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		cmd = exec.Command("openssl", "x509", "-req", "-in", ctxPath+"/madock.local.csr", "-CA", ctxPath+"/madockCA.pem", "-CAkey", ctxPath+"/madockCA.key", "-CAcreateserial", "-out", ctxPath+"/madock.local.crt", "-days", "365", "-sha256", "-extfile", ctxPath+"/madock.ca.ext")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		cmd = exec.Command("bash", "-c", "cat "+ctxPath+"/madock.local.crt "+ctxPath+"/madockCA.pem > "+ctxPath+"/fullchain.crt")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}
