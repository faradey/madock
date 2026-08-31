package nginx

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/v4/src/helper/configs/aruntime/proxytransform"
	"github.com/faradey/madock/v4/src/helper/finder"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/ports"
	"github.com/faradey/madock/v4/src/helper/setup/tools"
)

func MakeConf(projectName string) {
	// The shared proxy.conf lists every project's routing, so it must be
	// regenerated on every up — otherwise a freshly added/started project never
	// appears in the proxy until a rebuild clears the cache. The per-project
	// "<name>-proxy.conf" cache (used inside makeProxy) keeps this cheap: only
	// the current project's block is re-rendered, the rest are reused.
	//
	// conf-cache absent means a full reset is wanted (first start / rebuild /
	// clone remove it): drop the per-project caches so every block regenerates
	// from scratch. The proxy.go hash gate then reloads only if proxy.conf
	// actually changed, so regenerating on an unchanged start is a no-op.
	if !paths.IsFileExist(paths.CacheDir() + "/conf-cache") {
		cleanProxyCache()
	}

	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/nginx")
	setPorts(projectName)
	makeProxy(projectName)
	makeDockerfile(projectName)
	makeDockerCompose(projectName)
}

// cleanProxyCache removes old proxy config cache files
func cleanProxyCache() {
	cacheDir := paths.CacheDir()
	if paths.IsFileExist(cacheDir) {
		cacheFiles, _ := os.ReadDir(cacheDir)
		for _, f := range cacheFiles {
			if strings.HasSuffix(f.Name(), "-proxy.conf") {
				os.Remove(cacheDir + "/" + f.Name())
			}
		}
	}
}

func setPorts(projectName string) {
	// A project without a web server publishes nothing, so reserving the two
	// ports would hold them against every other project on the machine for a
	// container that is never created.
	if !configs2.NginxEnabledFor(projectName) {
		return
	}

	// Use the new ports package - it handles everything
	// Just ensure the project is registered
	_ = ports.GetPort(projectName, ports.ServiceNginx)
}

// proxyPreamble builds everything in the shared proxy configuration that comes
// before the per-project server blocks: worker settings, the http block, and the
// directives that must exist exactly once for all projects.
//
// A function of its configuration and nothing else, so the branches here can be
// asserted without generating a whole installation, which the golden file beside
// this package does end to end.
func proxyPreamble(generalConfig map[string]string) string {
	// worker_priority used to be set to -10 here and never worked: lowering a nice
	// value needs CAP_SYS_NICE, which is not in a container's default capability
	// set, so every start logged two `[alert] setpriority(-10) failed (13:
	// Permission denied)` — alert being the loudest level short of emerg, in the
	// one log where a real fault has to be visible. The proxy ran at ordinary
	// priority while the configuration claimed otherwise.
	//
	// Not worth a capability either: nginx here is epoll and I/O, and what starves
	// under load is php-fpm and the database, so a better-scheduled proxy only
	// accepts requests faster into the same queue. If priority is ever genuinely
	// wanted, the container-native knob is a cgroup weight — cpu_shares/cpus on the
	// proxy service in compose — which needs no extra privilege.
	preamble := "worker_processes 2;\nworker_rlimit_nofile 200000;\nevents {\n    worker_connections 4096;\nuse epoll;\n}\nhttp {\nserver_names_hash_bucket_size  128;\nserver_names_hash_max_size 1024;\n"

	// Global rate limiting zone (defined once for all projects)
	//
	// It was written to catch a request loop rather than an attacker, and the
	// default said so: 1000 requests a second from one address is a permission,
	// not a limit. The number moved; the zone did not need to.
	if generalConfig["proxy/rate_limit/enabled"] == "true" {
		preamble += "# Requests per second, per client address\nlimit_req_zone $binary_remote_addr zone=general:10m rate=" + generalConfig["proxy/rate_limit/rate"] + "r/s;\n"
	}

	// Simultaneous connections per address.
	//
	// The other half of resource exhaustion, and the half nothing answered: a
	// request that never finishes spends no rate at all, so a few hundred slow
	// connections hold every worker the proxy has while staying under any
	// per-second limit.
	if generalConfig["proxy/conn_limit/enabled"] == "true" {
		preamble += "# Simultaneous connections, per client address\nlimit_conn_zone $binary_remote_addr zone=addr:10m;\n"
	}

	// Global gzip settings (defined once for all projects)
	if generalConfig["proxy/gzip/enabled"] == "true" {
		preamble += "# Gzip compression\ngzip on;\ngzip_vary on;\ngzip_proxied any;\ngzip_comp_level 6;\ngzip_min_length 1000;\ngzip_types text/plain text/css text/xml application/json application/javascript application/xml+rss application/atom+xml image/svg+xml;\n"
	}

	// Global map for WebSocket upgrade. Used by Grafana Live AND by
	// the main `location /` proxy in default-proxy.conf so Next.js /
	// Vite / Rails Action Cable HMR sockets pass through unchanged
	// instead of being demoted to plain HTTP. The empty default keeps
	// `Connection:` empty for non-WS traffic so upstream keepalive
	// still works (close would force a new TCP per request).
	preamble += "# WebSocket upgrade map\nmap $http_upgrade $connection_upgrade {\n  default upgrade;\n  '' '';\n}\n"

	// Global log format and access log
	preamble += "# Access log format\nlog_format main '$remote_addr - $host [$time_local] \"$request\" '\n                '$status $body_bytes_sent \"$http_referer\" '\n                '\"$http_user_agent\" $request_time';\n"
	preamble += "access_log /var/log/nginx/access.log main;\n"

	return preamble
}

func makeProxy(projectName string) {
	generalConfig := configs2.GetGeneralConfig()
	/* Create nginx default configuration for Magento2 */
	nginxDefFile := ""
	allFileData := proxyPreamble(generalConfig)

	processedProjects := make(map[string]bool) // Track processed projects to avoid duplicates
	projectsNames := paths.GetDirs(paths.MakeDirsByPath(paths.RuntimeProjects()))
	if !finder.IsContain(projectsNames, projectName) {
		projectsNames = append(projectsNames, projectName)
	}

	// Pre-collect all domains to detect duplicates across all projects
	domainToProjects := make(map[string][]string)
	scannedProjects := make(map[string]bool)
	for _, name := range projectsNames {
		if scannedProjects[name] {
			continue
		}
		scannedProjects[name] = true
		if paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + name + "/config.xml") {
			projectConf := configs2.GetProjectConfig(name)
			// A project with no web server owns none of its domains: it is not
			// routed anywhere, so it cannot collide with a project that is.
			if !configs2.NginxEnabled(projectConf) {
				continue
			}
			hosts := configs2.GetHosts(projectConf)
			for _, hostAndStore := range hosts {
				domain := hostAndStore["name"]
				domainToProjects[domain] = append(domainToProjects[domain], name)
			}
		}
	}

	// Check for duplicate domains and report all projects that use them
	var duplicateErrors []string
	for domain, projects := range domainToProjects {
		if len(projects) > 1 {
			duplicateErrors = append(duplicateErrors, "Domain \""+domain+"\" is used in projects: "+strings.Join(projects, ", "))
		}
	}
	if len(duplicateErrors) > 0 {
		logger.Fatalln("Error. Duplicate domains found:\n" + strings.Join(duplicateErrors, "\n"))
	}

	// A registry entry whose source directory is gone keeps everything it had: its
	// ports stay reserved, and the block written below keeps routing its hosts at
	// containers that cannot exist. Said here because this is where the consequence
	// is created, and said rather than acted on: a stopped project keeps its routing
	// on purpose, and "the directory is not there right now" is also what an
	// unmounted disk looks like. Cleaning up is project:remove, deliberately.
	var withoutSource []string
	for _, entry := range configs2.ListProjects() {
		if entry.State == configs2.ProjectMissingSource {
			withoutSource = append(withoutSource, entry.Name)
		}
	}
	if len(withoutSource) > 0 {
		fmtc.WarningLn("These projects are still routed but their source directory is gone: " + strings.Join(withoutSource, ", "))
		fmtc.ToDoLn("Run madock project:list --stale")
	}

	for _, name := range projectsNames {
		// Skip if already processed (prevents duplicate upstream definitions)
		if processedProjects[name] {
			continue
		}
		processedProjects[name] = true
		pp := paths.NewProjectPaths(name)
		if paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + name + "/config.xml") {
			// A project with no web server has nothing to route to. Its cached
			// block is deleted rather than merely skipped: the cache is what a
			// project keeps while it is stopped, so leaving it there would put
			// the block back into proxy.conf on the next start of any other
			// project — which is exactly how a server_name for a project that
			// had its hosts removed survived on dev for weeks.
			if !configs2.NginxEnabledFor(name) {
				os.Remove(paths.CacheDir() + "/" + name + "-proxy.conf")
				continue
			}
			if !paths.IsFileExist(pp.StoppedFile()) {
				if projectName == name || !paths.IsFileExist(paths.CacheDir()+"/"+name+"-proxy.conf") {
					nginxDefFile = project.GetDockerConfigFile(name, "/nginx/conf/default-proxy.conf", "general")
					projectConf := configs2.GetProjectConfig(name)

					// The upstream is nginx, unless varnish is in front of it.
					upstreamPort := ports.GetPort(name, "nginx")
					if projectConf["varnish/enabled"] == "true" {
						upstreamPort = ports.GetPort(name, "varnish")
					}

					// HTTP/2 directive (new nginx 1.25+ syntax)
					http2Directive := ""
					if generalConfig["nginx/http/version"] == "http2" {
						http2Directive = "http2 on;"
					}

					// Rate limiting request directive (per-location, conditional)
					rateLimitReq := ""
					if generalConfig["proxy/rate_limit/enabled"] == "true" {
						rateLimitReq = "limit_req zone=general burst=" + generalConfig["proxy/rate_limit/burst"] + " nodelay;"
					}

					// And the connection limit beside it, so a block carries
					// both halves or neither.
					connLimit := ""
					if generalConfig["proxy/conn_limit/enabled"] == "true" {
						connLimit = "limit_conn addr " + generalConfig["proxy/conn_limit/per_ip"] + ";"
					}

					// The proxy block belongs to `name`, and until this was
					// rendered per project it was substituted with the config of
					// whichever project happened to be starting: the mftf
					// locations in one project's block followed another
					// project's setting.
					//
					// The listening ports and the timeouts are the proxy's own
					// and come from the general config, not from the project
					// whose block this is.
					strReplaced := project.Render(name, nginxDefFile, "nginx/conf/default-proxy.conf", map[string]string{
						"main_upstream_server":  "host.docker.internal:" + strconv.Itoa(upstreamPort),
						"nginx/port/unsecure":   generalConfig["nginx/port/unsecure"],
						"nginx/port/secure":     generalConfig["nginx/port/secure"],
						"nginx/http2/directive": http2Directive,
						"proxy/timeout/connect": generalConfig["proxy/timeout/connect"],
						"proxy/timeout/send":    generalConfig["proxy/timeout/send"],
						"proxy/timeout/read":    generalConfig["proxy/timeout/read"],
						"proxy/rate_limit/req":  rateLimitReq,
						"proxy/conn_limit/req":  connLimit,
						"proxy/max_body_size":   generalConfig["proxy/max_body_size"],
					})

					err := os.WriteFile(paths.MakeDirsByPath(paths.CacheDir())+"/"+name+"-proxy.conf", []byte(strReplaced), 0755)
					if err != nil {
						logger.Fatalln(err)
					}

					// Transformed here, where the project is known by name, and
					// not written to the cache in that form: the cache holds the
					// rendered block, the suffix is applied on the way into the
					// file so a transformer can change its mind without a stale
					// copy outliving it.
					allFileData += "\n" + proxytransform.ApplyProject(name, strReplaced)
				} else {
					strReplaced, err := os.ReadFile(paths.CacheDir() + "/" + name + "-proxy.conf")
					if err != nil {
						logger.Fatalln(err)
					}
					allFileData += "\n" + proxytransform.ApplyProject(name, string(strReplaced))
				}
			}
		}
	}

	// Build default server block with new http2 directive syntax (nginx 1.25+)
	http2DefaultDirective := ""
	if generalConfig["nginx/http/version"] == "http2" {
		http2DefaultDirective = "\n    http2 on;"
	}
	allFileData += "\nserver {\n    listen       " + generalConfig["nginx/port/unsecure"] + "  default_server;\n    listen " + generalConfig["nginx/port/secure"] + " default_server ssl;" + http2DefaultDirective + "\n    server_name  _;\n    return       444;\n    ssl_certificate /sslcert/fullchain.crt;\n    ssl_certificate_key /sslcert/madock.local.key;\n    include /sslcert/options-ssl-nginx.conf; \n}\n"
	allFileData += "\n}"
	nginxFile := paths.MakeDirsByPath(paths.CtxDir()) + "/proxy.conf"
	allFileData = proxytransform.Apply(allFileData)
	err := os.WriteFile(nginxFile, []byte(allFileData), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx default configuration for Magento2 */
}

func makeDockerfile(projectName string) {
	/* Create nginx Dockerfile configuration */
	ctxPath := paths.MakeDirsByPath(paths.CtxDir())
	nginxDefFile := paths.GetExecDirPath() + "/docker/general/nginx/proxy.Dockerfile"
	project.RenderTo(projectName, nginxDefFile, "general/nginx/proxy.Dockerfile", ctxPath+"/Dockerfile", nil)
	/* END Create nginx Dockerfile configuration */
}

func makeDockerCompose(projectName string) {
	/* Copy nginx docker-compose configuration */
	paths.MakeDirsByPath(paths.CtxDir())
	nginxDefFile := paths.GetExecDirPath() + "/docker/general/nginx/docker-compose-proxy.yml"
	project.RenderTo(projectName, nginxDefFile, "general/nginx/docker-compose-proxy.yml", paths.ProxyDockerCompose(), nil)
	/* END Create nginx Dockerfile configuration */
}

// sslAltNamesExt builds the openssl extension file that decides which hosts the
// certificate covers. It is derived from every registered project, so it
// changes whenever one is added, removed or has its hosts edited.
func sslAltNamesExt() string {
	projectsNames := paths.GetDirs(paths.MakeDirsByPath(paths.RuntimeProjects()))
	var commands []string
	i := 0
	for _, name := range projectsNames {
		if !paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + name + "/config.xml") {
			continue
		}

		projectConf := configs2.GetProjectConfig(name)
		// Nothing serves those names, so a certificate covering them would be
		// issued for a site that cannot answer.
		if !configs2.NginxEnabled(projectConf) {
			continue
		}
		hosts := configs2.GetHosts(projectConf)
		if len(hosts) == 0 {
			continue
		}

		for _, hostAndStore := range hosts {
			commands = append(commands, "DNS."+strconv.Itoa(i+2)+" = "+hostAndStore["name"])
			i++
		}
	}

	return "authorityKeyIdentifier=keyid,issuer\n" +
		"basicConstraints=CA:FALSE\n" +
		"keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment\n" +
		"subjectAltName = @alt_names\n" +
		"\n" +
		"[alt_names]\n" +
		"DNS.1 = madocklocalkey\n" +
		strings.Join(commands, "\n")
}

// SslCertCoversCurrentHosts reports whether the certificate on disk was issued
// for the projects that exist now.
//
// It matters because the certificate is shared by every project, while it used
// to be generated only when the proxy was started for the first time. A second
// project therefore got its route into proxy.conf — which is regenerated and
// reloaded on every start — and no certificate to go with it, so the browser
// refused the site that madock had just reported as running.
func SslCertCoversCurrentHosts(ctxPath string) bool {
	generalConfig := configs2.GetGeneralConfig()
	if val, ok := generalConfig["nginx/ssl/enabled"]; !ok || val != "true" {
		// Nothing to cover. Answering "yes" keeps the caller from regenerating
		// a certificate on every start of an installation that has SSL off.
		return true
	}

	existing, err := os.ReadFile(ctxPath + "/madock.ca.ext")
	if err != nil {
		return false
	}
	return string(existing) == sslAltNamesExt()
}

func GenerateSslCert(ctxPath string, force bool) {
	generalConfig := configs2.GetGeneralConfig()
	if val, ok := generalConfig["nginx/ssl/enabled"]; force || (ok && val == "true") {
		extFileContent := sslAltNamesExt()

		err := os.WriteFile(ctxPath+"/madock.ca.ext", []byte(extFileContent), 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}

		sslConfigFileContent := "ssl_session_cache shared:le_nginx_SSL:1m;\n" +
			"ssl_session_timeout 1440m;\n" +
			"\n" +
			"ssl_protocols TLSv1.2 TLSv1.3;\n" +
			"ssl_prefer_server_ciphers on;\n" +
			"\n" +
			"ssl_ciphers \"ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384\";"

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
			attr.AttachOutput(cmd)
			err = cmd.Run()
			if err != nil {
				logger.Fatal(err)
			}

			fmt.Println("Enter your password for adding an SSL certificate to your system.")

			if runtime.GOOS == "darwin" {
				cmd = exec.Command("sudo", "security", "delete-certificate", "-c", "madock")
				attr.AttachOutput(cmd)
				_ = cmd.Run()

				cmd = exec.Command("sudo", "security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", ctxPath+"/madockCA.pem")
				attr.AttachOutput(cmd)
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
				attr.AttachOutput(cmd)
				err = cmd.Run()
				if err != nil {
					logger.Fatal(err)
				}

				cmd = exec.Command("sudo", "chmod", "644", certPath+"/madockCA.crt")
				attr.AttachOutput(cmd)
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
					// -y reaches here too. Without this the flag stops being
					// true the moment a machine happens to lack certutil, and
					// `madock setup -y` in a provisioning script waits at a
					// prompt nobody will ever answer.
					askAboutCertutil := !tools.IsNonInteractive()

					if askAboutCertutil {
						fmt.Println("You need to install \"certutil\" to proceed with the certificate installation. Continue installation? y - continue. n - cancel certificate generation and continue without ssl.")
						fmt.Print("> ")
						buf := bufio.NewReader(os.Stdin)
						sentence, readErr := buf.ReadBytes('\n')
						if readErr != nil {
							// No terminal to answer with — a pipe, a cron job,
							// a CI step. Skipping SSL leaves a working project;
							// installing packages nobody agreed to does not, so
							// that is the safer reading of silence.
							fmtc.WarningLn("No answer possible on this input, continuing without SSL. Pass -y to install certutil instead.")
							selected = "n"
						} else {
							selected = strings.TrimSpace(string(sentence))
						}
					}

					if selected == "y" {
						cmd = exec.Command("sudo", "apt", "install", "-y", "libnss3-tools")
						attr.AttachOutput(cmd)
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
							attr.AttachOutput(cmd)
							_ = cmd.Run()
						}
					}
					cmd = exec.Command("certutil", "-d", "sql:"+usr.HomeDir+"/.pki/nssdb", "-A", "-t", "C,,", "-n", "madocklocalkey", "-i", ctxPath+"/madockCA.pem")
					attr.AttachOutput(cmd)
					err = cmd.Run()
					if err != nil {
						logger.Fatal(err)
					}
				}

				cmd = exec.Command("sudo", updateCertCommand...)
				attr.AttachOutput(cmd)
				err = cmd.Run()
				if err != nil {
					logger.Fatal(err)
				}
			}
		}

		cmd := exec.Command("openssl", "req", "-newkey", "rsa:4096", "-keyout", ctxPath+"/madock.local.key", "-out", ctxPath+"/madock.local.csr", "-nodes", "-subj", "/CN=madocklocalkey")
		attr.AttachOutput(cmd)
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}

		cmd = exec.Command("openssl", "x509", "-req", "-in", ctxPath+"/madock.local.csr", "-CA", ctxPath+"/madockCA.pem", "-CAkey", ctxPath+"/madockCA.key", "-CAcreateserial", "-out", ctxPath+"/madock.local.crt", "-days", "365", "-sha256", "-extfile", ctxPath+"/madock.ca.ext")
		attr.AttachOutput(cmd)
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}

		cmd = exec.Command("bash", "-c", "cat "+ctxPath+"/madock.local.crt "+ctxPath+"/madockCA.pem > "+ctxPath+"/fullchain.crt")
		attr.AttachOutput(cmd)
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}
}
