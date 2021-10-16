package project

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
)

func MakeConf(projectName string) {
	src := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/src"
	if _, err := os.Lstat(src); err == nil {
		if err := os.Remove(src); err != nil {
			log.Fatalf("failed to unlink: %+v", err)
		}
	}

	err := os.Symlink(paths.GetRunDirPath(), src)
	if err != nil {
		log.Fatal(err)
	}

	makeDockerCompose(projectName)
	makeNginxDockerfile(projectName)
	makeNginxConf(projectName)
	makePhpDockerfile(projectName)
	makeDBDockerfile(projectName)
	makeElasticDockerfile(projectName)
	makeRedisDockerfile(projectName)
	makeNodeDockerfile(projectName)
}

func makeNginxDockerfile(projectName string) {
	dockerDefFile := getDockerFile(projectName, "/docker/nginx/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	usr, err := user.Current()
	if err == nil {
		str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
		str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
	} else {
		log.Fatal(err)
	}

	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/nginx.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNginxConf(projectName string) {
	defFile := getDockerFile(projectName, "/docker/nginx/conf/default.conf")

	b, err := os.ReadFile(defFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	projectConf := configs.GetProjectConfig()
	str = strings.Replace(str, "{{{NGINX_PORT}}}", projectConf["NGINX_PORT"], -1)
	hostName := "loc." + projectName + ".com"
	hostNameWebsites := "loc." + projectName + ".com base;"
	if val, ok := projectConf["HOSTS"]; ok {
		var onlyHosts []string
		var websitesHosts []string
		hosts := strings.Split(val, " ")
		if len(hosts) > 0 {
			for _, hostAndStore := range hosts {
				onlyHosts = append(onlyHosts, strings.Split(hostAndStore, ":")[0])
				if len(strings.Split(hostAndStore, ":")) > 1 {
					websitesHosts = append(websitesHosts, strings.Split(hostAndStore, ":")[0]+" "+strings.Split(hostAndStore, ":")[1]+";")
				}
			}
			if len(onlyHosts) > 0 {
				hostName = strings.Join(onlyHosts, "\n")
			}
			if len(websitesHosts) > 0 {
				hostNameWebsites = strings.Join(websitesHosts, "\n")
			}
		}
	}
	str = strings.Replace(str, "{{{HOST_NAMES}}}", hostName, -1)
	str = strings.Replace(str, "{{{PROJECT_NAME}}}", projectName, -1)
	str = strings.Replace(str, "{{{HOST_NAMES_WEBSITES}}}", hostNameWebsites, -1)

	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx")
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/nginx.conf"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makePhpDockerfile(projectName string) {
	dockerDefFile := getDockerFile(projectName, "/docker/php/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	projectConf := configs.GetProjectConfig()
	str := string(b)
	str = strings.Replace(str, "{{{PHP_VERSION}}}", projectConf["PHP_VERSION"], -1)
	str = strings.Replace(str, "{{{PHP_TZ}}}", projectConf["PHP_TZ"], -1)
	str = strings.Replace(str, "{{{PHP_MODULE_XDEBUG}}}", projectConf["PHP_MODULE_XDEBUG"], -1)
	str = strings.Replace(str, "{{{PHP_XDEBUG_REMOTE_HOST}}}", projectConf["PHP_XDEBUG_REMOTE_HOST"], -1)
	str = strings.Replace(str, "{{{PHP_MODULE_IONCUBE}}}", projectConf["PHP_MODULE_IONCUBE"], -1)
	str = strings.Replace(str, "{{{PHP_COMPOSER_VERSION}}}", projectConf["PHP_COMPOSER_VERSION"], -1)
	usr, err := user.Current()
	if err == nil {
		str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
		str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
	} else {
		log.Fatal(err)
	}
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/php.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeDockerCompose(projectName string) {
	dockerDefFile := getDockerFile(projectName, "/docker/docker-compose.yml")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	portsConfig := configs.ParseFile(paths.GetExecDirPath() + "/aruntime/ports.conf")
	portNumber, err := strconv.Atoi(portsConfig[projectName])
	if err != nil {
		log.Fatal(err)
	}

	portNumberRanged := (portNumber - 1) * 20
	hostName := "loc." + projectName + ".com"
	projectConf := configs.GetProjectConfig()
	if val, ok := projectConf["HOSTS"]; ok {
		hosts := strings.Split(val, " ")
		if len(hosts) > 0 {
			hostName = strings.Split(hosts[0], ":")[0]
		}
	}
	str = strings.Replace(str, "{{{RABBITMQ_VERSION}}}", projectConf["RABBITMQ_VERSION"], -1)
	str = strings.Replace(str, "{{{HOST_NAME_DEFAULT}}}", hostName, -1)
	str = strings.Replace(str, "{{{NGINX_PORT}}}", strconv.Itoa(portNumberRanged+17000), -1)
	for i := 1; i < 20; i++ {
		str = strings.Replace(str, "{{{NGINX_PORT+"+strconv.Itoa(i)+"}}}", strconv.Itoa(portNumberRanged+17000+i), -1)
	}
	str = strings.Replace(str, "{{{NETWORK_NUMBER}}}", strconv.Itoa(portNumber+90), -1)

	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/docker-compose.yml"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeDBDockerfile(projectName string) {
	dockerDefFile := getDockerFile(projectName, "/docker/db/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	projectConf := configs.GetProjectConfig()
	str := string(b)
	str = strings.Replace(str, "{{{DB_VERSION}}}", projectConf["DB_VERSION"], -1)
	str = strings.Replace(str, "{{{DB_ROOT_PASSWORD}}}", projectConf["DB_ROOT_PASSWORD"], -1)
	str = strings.Replace(str, "{{{DB_DATABASE}}}", projectConf["DB_DATABASE"], -1)
	str = strings.Replace(str, "{{{DB_USER}}}", projectConf["DB_USER"], -1)
	str = strings.Replace(str, "{{{DB_PASSWORD}}}", projectConf["DB_PASSWORD"], -1)

	usr, err := user.Current()
	if err == nil {
		str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
		str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
	} else {
		log.Fatal(err)
	}
	nginxFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx") + "/db.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}

	myCnfFile := paths.GetExecDirPath() + "/docker/db/my.cnf"
	if _, err := os.Stat(myCnfFile); os.IsNotExist(err) {
		log.Fatal(err)
	}

	b, err = os.ReadFile(myCnfFile)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx/my.cnf", b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeElasticDockerfile(projectName string) {
	dockerDefFile := getDockerFile(projectName, "/docker/elasticsearch/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	projectConf := configs.GetProjectConfig()

	str := string(b)
	str = strings.Replace(str, "{{{ELASTICSEARCH_VERSION}}}", projectConf["ELASTICSEARCH_VERSION"], -1)
	usr, err := user.Current()
	if err == nil {
		str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
		str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
	} else {
		log.Fatal(err)
	}
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/elasticsearch.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeRedisDockerfile(projectName string) {
	dockerDefFile := getDockerFile(projectName, "/docker/redis/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	projectConf := configs.GetProjectConfig()

	str := string(b)
	str = strings.Replace(str, "{{{REDIS_VERSION}}}", projectConf["REDIS_VERSION"], -1)
	usr, err := user.Current()
	if err == nil {
		str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
		str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
	} else {
		log.Fatal(err)
	}
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/redis.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func makeNodeDockerfile(projectName string) {
	dockerDefFile := getDockerFile(projectName, "/docker/node/Dockerfile")

	b, err := os.ReadFile(dockerDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	projectConf := configs.GetProjectConfig()
	str = strings.Replace(str, "{{{PHP_VERSION}}}", projectConf["PHP_VERSION"], -1)
	usr, err := user.Current()
	if err == nil {
		str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
		str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
	} else {
		log.Fatal(err)
	}
	nginxFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ctx/node.Dockerfile"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func getDockerFile(projectName, path string) string {
	dockerDefFile := paths.GetExecDirPath() + "/projects/" + projectName + path
	if _, err := os.Stat(dockerDefFile); os.IsNotExist(err) {
		dockerDefFile = paths.GetExecDirPath() + path
		if _, err = os.Stat(dockerDefFile); os.IsNotExist(err) {
			log.Fatal(err)
		}
	}

	return dockerDefFile
}
