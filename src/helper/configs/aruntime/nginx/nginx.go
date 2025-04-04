package nginx

import (
	"bufio"
	"bytes"
	"fmt"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/src/helper/finder"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func MakeConf(projectName string) {
	if paths.IsFileExist(paths.GetExecDirPath() + "/cache/conf-cache") {
		return
	}
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/nginx")
	setPorts(projectName)
	makeProxy(projectName)
	makeDockerfile(projectName)
	makeDockerCompose(projectName)
}

func setPorts(projectName string) {
	projectsAruntime := paths.GetDirs(paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects"))
	projects := paths.GetDirs(paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects"))
	if len(projectsAruntime) > len(projects) {
		projects = projectsAruntime
	}
	portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
	if !paths.IsFileExist(portsFile) {
		lines := ""
		for port, line := range projects {
			lines += line + "=" + strconv.Itoa(port+1) + "\n"
		}
		_ = os.WriteFile(portsFile, []byte(lines), 0664)
	}

	portsConfig := configs2.ParseFile(portsFile)
	lines := ""
	for projName, port := range portsConfig {
		if paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + projName) {
			lines += projName + "=" + port + "\n"
		}
	}

	if lines != "" {
		_ = os.WriteFile(portsFile, []byte(lines), 0664)
	}

	if _, ok := portsConfig[projectName]; !ok {
		f, err := os.OpenFile(portsFile,
			os.O_APPEND|os.O_WRONLY, 0664)
		if err != nil {
			log.Println(err)
		}
		defer f.Close()
		maxPort := getMaxPort(portsConfig)
		if _, err := f.WriteString(projectName + "=" + strconv.Itoa(maxPort+1) + "\n"); err != nil {
			log.Println(err)
		}
	}
}

func makeProxy(projectName string) {
	portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
	portsConfig := configs2.ParseFile(portsFile)
	generalConfig := configs2.GetGeneralConfig()
	/* Create nginx default configuration for Magento2 */
	nginxDefFile := ""
	str := ""
	allFileData := "worker_processes 2;\nworker_priority -10;\nworker_rlimit_nofile 200000;\nevents {\n    worker_connections 4096;\nuse epoll;\n}\nhttp {\nserver_names_hash_bucket_size  128;\nserver_names_hash_max_size 1024;\n"

	var onlyHostsGlobal []string
	projectsNames := paths.GetDirs(paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects"))
	if !finder.IsContain(projectsNames, projectName) {
		projectsNames = append(projectsNames, projectName)
	}
	for _, name := range projectsNames {
		if paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + name + "/config.xml") {
			if !paths.IsFileExist(paths.GetExecDirPath() + "/aruntime/projects/" + name + "/stopped") {
				if projectName == name || !paths.IsFileExist(paths.GetExecDirPath()+"/cache/"+name+"-proxy.conf") {
					nginxDefFile = project.GetDockerConfigFile(name, "/nginx/conf/default-proxy.conf", "general")
					b, err := os.ReadFile(nginxDefFile)
					if err != nil {
						logger.Fatal(err)
					}

					str = string(b)
					port, err := strconv.Atoi(portsConfig[name])
					if err != nil {
						fmt.Println("Project name is " + name)
						logger.Fatal(err)
					}
					projectConf := configs2.GetProjectConfig(name)
					portRanged := (port - 1) * 20
					strReplaced := ""
					if projectConf["varnish/enabled"] != "true" {
						strReplaced = strings.Replace(str, "{{{nginx/port/default}}}", strconv.Itoa(17000+portRanged), -1)
					} else {
						strReplaced = strings.Replace(str, "{{{nginx/port/default}}}", strconv.Itoa(17000+portRanged+9), -1)
					}
					strReplaced = strings.Replace(strReplaced, "{{{nginx/port/unsecure}}}", generalConfig["nginx/port/unsecure"], -1)
					strReplaced = strings.Replace(strReplaced, "{{{nginx/port/secure}}}", generalConfig["nginx/port/secure"], -1)
					strReplaced = strings.Replace(strReplaced, "{{{nginx/http/version}}}", generalConfig["nginx/http/version"], -1)
					for i := 1; i < 20; i++ {
						strReplaced = strings.Replace(strReplaced, "{{{nginx/port/default+"+strconv.Itoa(i)+"}}}", strconv.Itoa(17000+portRanged+i), -1)
					}
					strReplaced = configs2.ReplaceConfigValue(projectName, strReplaced)
					hostName := "loc." + name + ".com"
					hosts := configs2.GetHosts(projectConf)
					if len(hosts) > 0 {
						var onlyHosts []string
						domain := ""
						for _, hostAndStore := range hosts {
							domain = hostAndStore["name"]
							if finder.IsContain(onlyHostsGlobal, domain) {
								logger.Fatalln("Error. Duplicate domain " + domain)
							}
							onlyHosts = append(onlyHosts, domain)
							onlyHostsGlobal = append(onlyHostsGlobal, domain)
						}
						hostName = strings.Join(onlyHosts, "\n")
					}

					strReplaced = strings.Replace(strReplaced, "{{{nginx/host_names}}}", hostName, -1)

					err = os.WriteFile(paths.MakeDirsByPath(paths.GetExecDirPath()+"/cache/")+name+"-proxy.conf", []byte(strReplaced), 0755)
					if err != nil {
						logger.Fatalln(err)
					}

					allFileData += "\n" + strReplaced
				} else {
					strReplaced, err := os.ReadFile(paths.GetExecDirPath() + "/cache/" + name + "-proxy.conf")
					if err != nil {
						logger.Fatalln(err)
					}
					allFileData += "\n" + string(strReplaced)
				}
			}
		}
	}

	allFileData += "\nserver {\n    listen       " + generalConfig["nginx/port/unsecure"] + "  default_server;\n listen " + generalConfig["nginx/port/secure"] + " default_server ssl " + generalConfig["nginx/http/version"] + ";\n    server_name  _;\n    return       444;\n ssl_certificate /sslcert/fullchain.crt;\n        ssl_certificate_key /sslcert/madock.local.key;\n        include /sslcert/options-ssl-nginx.conf; \n}\n"
	allFileData += "\n}"
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/ctx") + "/proxy.conf"
	err := os.WriteFile(nginxFile, []byte(allFileData), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx default configuration for Magento2 */
}

func makeDockerfile(projectName string) {
	/* Create nginx Dockerfile configuration */
	ctxPath := paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/general/nginx/proxy.Dockerfile"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		logger.Fatal(err)
	}

	str := string(b)
	str = configs2.ReplaceConfigValue(projectName, str)

	err = os.WriteFile(ctxPath+"/Dockerfile", []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}

func makeDockerCompose(projectName string) {
	/* Copy nginx docker-compose configuration */
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/general/nginx/docker-compose-proxy.yml"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		logger.Fatal(err)
	}

	str := string(b)
	str = configs2.ReplaceConfigValue(projectName, str)

	err = os.WriteFile(paths.GetExecDirPath()+"/aruntime/docker-compose.yml", []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}

func getMaxPort(conf map[string]string) int {
	portInt := 0
	var err error
	var ports []int
	for _, port := range conf {
		portInt, err = strconv.Atoi(port)
		if err != nil {
			logger.Fatal(err)
		}
		ports = append(ports, portInt)
	}

	for i := 1; i < 1000; i++ {
		if !finder.IsContainInt(ports, i) {
			return i - 1
		}
	}

	return 0
}

func GenerateSslCert(ctxPath string, force bool) {
	generalConfig := configs2.GetGeneralConfig()
	if val, ok := generalConfig["nginx/ssl/enabled"]; force || (ok && val == "true") {
		projectsNames := paths.GetDirs(paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects"))
		var commands []string
		i := 0
		for _, name := range projectsNames {
			if !paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + name + "/config.xml") {
				continue
			}

			projectConf := configs2.GetProjectConfig(name)
			hosts := configs2.GetHosts(projectConf)
			if len(hosts) == 0 {
				continue
			}

			for _, hostAndStore := range hosts {
				commands = append(commands, "DNS."+strconv.Itoa(i+2)+" = "+hostAndStore["name"])
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
		if !paths.IsFileExist(ctxPath + "/madockCA.pem") {
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
				logger.Fatal(err)
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
					logger.Fatal(err)
				}
			} else if runtime.GOOS == "linux" {
				content, err := os.ReadFile("/etc/os-release")
				if err != nil {
					logger.Fatal(err)
				}

				osRelease := string(content)
				var certPath string
				var updateCertCommand []string

				distroPaths := map[string]string{
					"Arch Linux": "/etc/ca-certificates/trust-source/anchors",
					"default":    "/usr/local/share/ca-certificates",
				}

				distroUpdateCert := map[string][]string{
					"Arch Linux": {"update-ca-trust"},
					"default":    {"update-ca-certificates", "-f"},
				}

				if strings.Contains(osRelease, "Arch Linux") {
					certPath = distroPaths["Arch Linux"]
					updateCertCommand = distroUpdateCert["Arch Linux"]
				} else {
					certPath = distroPaths["default"]
					updateCertCommand = distroUpdateCert["default"]
				}

				cmd = exec.Command("sudo", "cp", ctxPath+"/madockCA.pem", certPath+"/madockCA.crt")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					logger.Fatal(err)
				}

				cmd = exec.Command("sudo", "chmod", "644", certPath+"/madockCA.crt")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					logger.Fatal(err)
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
						logger.Fatalln(err)
					}
					selected = strings.TrimSpace(string(sentence))
					if selected == "y" {
						cmd = exec.Command("sudo", "apt", "install", "-y", "libnss3-tools")
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						err = cmd.Run()
						if err != nil {
							logger.Fatal(err)
						}
					}
				}

				if selected == "y" {
					usr, _ := user.Current()
					if !paths.IsFileExist(usr.HomeDir + "/.pki/nssdb") {
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
						logger.Fatal(err)
					}
				}

				cmd = exec.Command("sudo", updateCertCommand...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					logger.Fatal(err)
				}
			}
		}

		cmd := exec.Command("openssl", "req", "-newkey", "rsa:4096", "-keyout", ctxPath+"/madock.local.key", "-out", ctxPath+"/madock.local.csr", "-nodes", "-subj", "/CN=madocklocalkey")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}

		cmd = exec.Command("openssl", "x509", "-req", "-in", ctxPath+"/madock.local.csr", "-CA", ctxPath+"/madockCA.pem", "-CAkey", ctxPath+"/madockCA.key", "-CAcreateserial", "-out", ctxPath+"/madock.local.crt", "-days", "365", "-sha256", "-extfile", ctxPath+"/madock.ca.ext")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}

		cmd = exec.Command("bash", "-c", "cat "+ctxPath+"/madock.local.crt "+ctxPath+"/madockCA.pem > "+ctxPath+"/fullchain.crt")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}
}
