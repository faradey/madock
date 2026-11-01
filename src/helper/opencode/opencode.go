// Package opencode puts the opencode CLI into a project as a sandbox.
//
// opencode's own `permission.external_directory: deny` stops its write tool
// at the project directory and nothing else: a shell command it runs writes
// wherever the user can. Measured on the host before this was built. So the
// boundary is the container, not the configuration, and the container is cut
// down to what the work needs:
//
//   - opencode: node, the opencode CLI and an ssh client. It mounts the
//     project's code and sits on one network, which is internal — no
//     database, no cache, no internet.
//   - opencode-llm: nginx forwarding to opencode/base_url and nowhere else.
//     The only service on that network with a way out.
//   - opencode-exec: the project's own image with sshd, for php, composer,
//     bin/magento, go and the rest. opencode reaches it through shims of the
//     same names. It is on the project's network as well, because those tools
//     need the database; what runs here reaches whatever the application does.
//
// Measured on a stand on 2026-11-01 with the same layout: the opencode
// container reached the model through the proxy and got curl exit 6 for the
// internet, the host and the database.
package opencode

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// Service is the container `madock opencode` runs in.
	Service = "opencode"
	// ExecService runs the project's tools for it, over ssh.
	ExecService = "opencode-exec"
	// LLMService forwards to the model.
	LLMService = "opencode-llm"

	network       = "opencode"
	egressNetwork = "opencode-egress"
	dataVolume    = "opencodedata"
	llmPort       = "8080"
	proxyImage    = "nginx:1.27-alpine"

	// ConfDir is where the generated files are mounted, in both containers.
	ConfDir = "/etc/madock-opencode"

	// Ctx file names, in the project's ctx/.
	ConfigCtxFile = "opencode.json"
	AgentsCtxFile = "opencode-AGENTS.md"
	SshdCtxFile   = "opencode-sshd.conf"
	LLMCtxFile    = "opencode-llm.conf"

	// marker keeps the Dockerfile transform idempotent and names the block.
	marker = "madock: opencode-exec"
)

// Enabled says whether the project runs opencode.
func Enabled(conf map[string]string) bool {
	return conf["opencode/enabled"] == "true"
}

// installBlock goes into the application image. The application container
// never starts sshd — the package is files it does not execute; the daemon
// runs only in opencode-exec, whose command is the script below.
//
// The account shares www-data's uid, so files it writes belong to the
// application, and has a shell, which www-data does not and sshd requires.
// `-p '*'` rather than the default `!`: with PAM off, OpenSSH reads `!` as a
// locked account and refuses even a key.
//
// sshd gives a session its own PATH, not the image's, and the image's
// environment is what finds go in /usr/local/go/bin or php's ini directory.
// So the environment the container started with is handed to every session
// through SetEnv, one directive, since sshd keeps the first value it reads.
const installBlock = "\n" +
	"# " + marker + " (sshd runs only in the opencode-exec service)\n" +
	"RUN apt-get update && apt-get install -y --no-install-recommends openssh-server \\\n" +
	"    && rm -rf /var/lib/apt/lists/* && mkdir -p /run/sshd\n" +
	"RUN cat > /usr/local/bin/madock-opencode-sshd <<'MADOCK_EOF' && chmod 755 /usr/local/bin/madock-opencode-sshd\n" +
	"#!/bin/sh\n" +
	"getent passwd opencode >/dev/null || useradd -o -u \"$(id -u www-data)\" -g \"$(id -g www-data)\" -p '*' -M -d /var/www -s /bin/bash opencode\n" +
	"{\n" +
	"    printf 'SetEnv'\n" +
	"    env | grep -E '^[A-Za-z_][A-Za-z0-9_]*=' | grep -v -E '^(HOME|HOSTNAME|PWD|SHLVL|_|TERM|USER|LOGNAME)=' | grep -v '\"' | while IFS= read -r line; do printf ' \"%s\"' \"$line\"; done\n" +
	"    echo\n" +
	"} > /etc/ssh/sshd_config.d/01-madock-opencode-env.conf\n" +
	"mkdir -p /run/sshd\n" +
	"exec /usr/sbin/sshd -D -e\n" +
	"MADOCK_EOF\n"

// InjectDockerfile appends the sshd block to an application image.
//
// Some images end as their own user — Spree's as ruby, Saleor's as saleor —
// and apt cannot install as either. The block runs as root and gives the
// image its last USER back, so the application container runs as it did.
func InjectDockerfile(content string) string {
	if strings.Contains(content, marker) {
		return content
	}
	user := ""
	for _, l := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "USER ") {
			user = strings.TrimSpace(l)
		}
	}
	block := installBlock
	if user != "" {
		header, rest, _ := strings.Cut(strings.TrimPrefix(block, "\n"), "\n")
		block = "\n" + header + "\nUSER root\n" + rest + user + "\n"
	}
	return strings.TrimRight(content, "\n") + "\n" + block
}

// MainDockerfile says whether fileName is the image of the project's main
// service — the one opencode-exec is built from. Only that image gets sshd: a
// php project's node container is built from nodejs.Dockerfile as well, and
// has no use for it.
func MainDockerfile(mainService, fileName string) bool {
	switch mainService {
	case "php":
		return fileName == "php.Dockerfile" || fileName == "php.DockerfileWithoutXdebug"
	case "nodejs", "python", "golang", "ruby", "app":
		return fileName == mainService+".Dockerfile"
	}
	return false
}

// InjectCompose writes the three services, their networks and the data
// volume into a rendered compose file.
//
// opencode-exec takes four things from the main service — build, the code
// volume, extra_hosts and networks — and writes the rest itself. That is an
// allowlist, not a copy with exclusions: the main service also mounts the
// host user's ~/.ssh and ~/.composer, and carries an environment other code
// injects secrets into, and none of that belongs where a model runs commands.
//
// Returns the content unchanged when the main service is not in the file;
// half a service there would stop the whole stack from starting.
func InjectCompose(content, mainService string) string {
	if strings.Contains(content, "\n  "+Service+":\n") {
		return content
	}
	block := serviceBlock(content, mainService)
	if block == nil {
		return content
	}
	code := codeMount(block)
	if code == "" {
		return content
	}
	restart := line(block, "restart")

	var svc []string
	svc = append(svc,
		"  "+Service+":",
		"    init: true",
		"    build:",
		"      context: ctx",
		"      dockerfile: opencode.Dockerfile",
		"    tty: true",
		"    volumes:",
		code,
		"      - ./ctx/"+ConfigCtxFile+":"+ConfDir+"/opencode.json:ro",
		"      - ./ctx/"+AgentsCtxFile+":"+ConfDir+"/AGENTS.md:ro",
		"      - ./"+keysDir+"/"+clientKeyFile+":"+ConfDir+"/id_ed25519:ro",
		"      - ./"+keysDir+"/"+knownHostsFile+":"+ConfDir+"/known_hosts:ro",
		"      - "+dataVolume+":/var/www/.local/share/opencode",
		"    environment:",
		"      - OPENCODE_CONFIG="+ConfDir+"/opencode.json",
		"    networks:",
		"      - "+network,
		"    depends_on:",
		"      - "+LLMService,
		"      - "+ExecService,
	)
	svc = appendLine(svc, restart)

	svc = append(svc, "  "+ExecService+":", "    init: true")
	svc = append(svc, section(block, "build")...)
	svc = append(svc,
		// Root for useradd and sshd, whatever user the image ends as.
		"    user: root",
		`    command: ["/usr/local/bin/madock-opencode-sshd"]`,
		"    volumes:",
		code,
		"      - ./ctx/"+SshdCtxFile+":/etc/ssh/sshd_config.d/00-madock-opencode.conf:ro",
		"      - ./"+keysDir+"/"+hostKeyFile+":"+ConfDir+"/hostkey:ro",
		"      - ./"+keysDir+"/"+authorizedKeysFile+":"+ConfDir+"/authorized_keys:ro",
	)
	svc = append(svc, section(block, "extra_hosts")...)
	svc = append(svc, "    networks:")
	if nets := section(block, "networks"); len(nets) > 0 {
		svc = append(svc, nets[1:]...)
	} else {
		// A service with no networks key is on the default network, and
		// naming one drops it from there — so it is named.
		svc = append(svc, "      - default")
	}
	svc = append(svc,
		"      - "+network,
		// no-new-privileges stops a setuid binary in the image — sudo, su —
		// from granting anything to the session. sshd itself starts as root
		// and only ever drops.
		"    security_opt:",
		"      - no-new-privileges:true",
	)
	svc = appendLine(svc, restart)

	svc = append(svc,
		"  "+LLMService+":",
		"    init: true",
		"    image: "+proxyImage,
		"    volumes:",
		"      - ./ctx/"+LLMCtxFile+":/etc/nginx/conf.d/default.conf:ro",
		"    extra_hosts:",
		`      - "host.docker.internal:host-gateway"`,
		"    networks:",
		"      - "+network,
		"      - "+egressNetwork,
	)
	svc = appendLine(svc, restart)

	lines := strings.Split(content, "\n")
	var out []string
	services, volumes, networks := false, false, false
	for _, l := range lines {
		switch strings.TrimRight(l, " \t") {
		case "volumes:":
			if !services {
				out = append(out, svc...)
				services = true
			}
			out = append(out, l, "  "+dataVolume+":")
			volumes = true
			continue
		case "networks:":
			if !services {
				out = append(out, svc...)
				services = true
			}
			out = append(out, l,
				"  "+network+":",
				"    driver: bridge",
				"    internal: true",
				"  "+egressNetwork+":",
				"    driver: bridge",
			)
			networks = true
			continue
		}
		out = append(out, l)
	}
	if !services {
		return content
	}
	if !volumes {
		out = append(out, "", "volumes:", "  "+dataVolume+":")
	}
	if !networks {
		out = append(out, "", "networks:",
			"  "+network+":",
			"    driver: bridge",
			"    internal: true",
			"  "+egressNetwork+":",
			"    driver: bridge",
		)
	}
	return strings.Join(out, "\n")
}

func appendLine(lines []string, l string) []string {
	if l == "" {
		return lines
	}
	return append(lines, l)
}

// serviceBlock returns the lines of one top-level service, header included,
// or nil when it is not in the file.
func serviceBlock(content, name string) []string {
	lines := strings.Split(content, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimRight(l, " \t") == "  "+name+":" {
			start = i
			break
		}
	}
	if start < 0 {
		return nil
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			continue
		}
		if !strings.HasPrefix(l, "    ") && !strings.HasPrefix(l, "  #") {
			end = i
			break
		}
	}
	return lines[start:end]
}

// section returns a service's `    key:` line and everything indented under
// it, comments dropped.
func section(block []string, key string) []string {
	var out []string
	in := false
	for _, l := range block {
		if strings.TrimRight(l, " \t") == "    "+key+":" {
			in = true
			out = append(out, l)
			continue
		}
		if !in {
			continue
		}
		if strings.TrimSpace(l) == "" || strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		if !strings.HasPrefix(l, "      ") {
			break
		}
		out = append(out, l)
	}
	return out
}

// line returns a service's single-line `    key: value` entry, or "".
func line(block []string, key string) string {
	for _, l := range block {
		if strings.HasPrefix(l, "    "+key+":") {
			return l
		}
	}
	return ""
}

// codeMount is the main service's bind of the project source, recognised by
// its source, `./src`. Both new containers mount it at the same target, so a
// directory in one is the same directory in the other — which is what lets a
// shim run a tool where opencode stands.
func codeMount(block []string) string {
	for _, l := range block {
		if strings.HasPrefix(strings.TrimSpace(l), "- ./src:") {
			return l
		}
	}
	return ""
}

// Upstream splits base_url into what nginx forwards to and what opencode
// asks for: scheme and host go to the proxy, the path stays with the client.
func Upstream(baseURL string) (origin, path string, err error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", "", fmt.Errorf("opencode/base_url must be an http or https URL with a host, got %q", baseURL)
	}
	return u.Scheme + "://" + u.Host, strings.TrimRight(u.EscapedPath(), "/"), nil
}

// LLMConf is the proxy's whole configuration: one server, one upstream.
func LLMConf(baseURL string) (string, error) {
	origin, _, err := Upstream(baseURL)
	if err != nil {
		return "", err
	}
	host := strings.SplitN(strings.SplitN(origin, "://", 2)[1], "/", 2)[0]
	lines := []string{
		"# generated by madock for " + LLMService + ": the only way out of the " + network + " network; edits here are overwritten",
		"server {",
		"    listen " + llmPort + ";",
		"    location / {",
		"        proxy_pass " + origin + ";",
		"        proxy_http_version 1.1;",
		"        proxy_set_header Host " + host + ";",
	}
	if strings.HasPrefix(origin, "https://") {
		lines = append(lines, "        proxy_ssl_server_name on;")
	}
	lines = append(lines,
		// Answers stream token by token, and a long one takes minutes.
		"        proxy_buffering off;",
		"        proxy_read_timeout 3600s;",
		"        proxy_send_timeout 3600s;",
		"        client_max_body_size 64m;",
		"    }",
		"}",
		"",
	)
	return strings.Join(lines, "\n"), nil
}

// Config is opencode.json: one provider, the proxy, and the model.
func Config(conf map[string]string) (string, error) {
	_, path, err := Upstream(conf["opencode/base_url"])
	if err != nil {
		return "", err
	}
	model := strings.TrimSpace(conf["opencode/model"])
	if model == "" {
		return "", fmt.Errorf("opencode/model is empty")
	}
	limits := map[string]int{}
	for _, key := range []string{"context", "output"} {
		n, err := strconv.Atoi(strings.TrimSpace(conf["opencode/"+key]))
		if err != nil || n <= 0 {
			return "", fmt.Errorf("opencode/%s must be a positive number, got %q", key, conf["opencode/"+key])
		}
		limits[key] = n
	}

	type modelConf struct {
		Name  string         `json:"name"`
		Limit map[string]int `json:"limit"`
	}
	type provider struct {
		NPM     string               `json:"npm"`
		Name    string               `json:"name"`
		Options map[string]string    `json:"options"`
		Models  map[string]modelConf `json:"models"`
	}
	doc := struct {
		Schema       string              `json:"$schema"`
		Provider     map[string]provider `json:"provider"`
		Model        string              `json:"model"`
		Instructions []string            `json:"instructions"`
		Permission   map[string]string   `json:"permission"`
		Autoupdate   bool                `json:"autoupdate"`
		Share        string              `json:"share"`
	}{
		Schema: "https://opencode.ai/config.json",
		Provider: map[string]provider{
			"madock": {
				NPM:     "@ai-sdk/openai-compatible",
				Name:    "madock",
				Options: map[string]string{"baseURL": "http://" + LLMService + ":" + llmPort + path},
				Models:  map[string]modelConf{model: {Name: model, Limit: limits}},
			},
		},
		Model:        "madock/" + model,
		Instructions: []string{ConfDir + "/AGENTS.md"},
		Permission:   map[string]string{"external_directory": "deny"},
		// Neither can work from a network with no way out, and a failed
		// attempt is noise on every start.
		Autoupdate: false,
		Share:      "disabled",
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out) + "\n", nil
}

// DefaultAgents is what the model is told when the project names no file of
// its own. It says what the container is, because nothing else will: without
// it a model looks for php, finds the shim, and has no idea why the database
// answers from a container it cannot see.
const DefaultAgents = `# Working in this container

- This container holds the project's files and nothing else. php, composer,
  go, python, ruby and the project's own scripts run in the project's main
  container through commands of the same name, in the same directory.
  ` + "`madock-exec <command> [args...]`" + ` runs any other command there.
- Work only inside the project directory.
- After every edit, check it: a syntax check of the file you changed, and the
  project's tests if it has them. Fix what fails before saying the task is done.
- Say plainly when something failed or was not checked.
- Do not commit or push unless asked.
`

// Agents returns the instructions file: the project's own when
// opencode/agents_file names one, the default otherwise. A named file that
// cannot be read is an error, not a fallback — a sandbox quietly running
// with instructions other than the ones somebody wrote is the wrong default.
func Agents(conf map[string]string) (string, error) {
	path := strings.TrimSpace(conf["opencode/agents_file"])
	if path == "" {
		return DefaultAgents, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("opencode/agents_file: %w", err)
	}
	return string(raw), nil
}

// SshdConf is the drop-in for opencode-exec. One account, one key, nothing
// forwarded.
func SshdConf() string {
	return strings.Join([]string{
		"# generated by madock for " + ExecService + "; edits here are overwritten",
		"Port 22",
		"HostKey " + ConfDir + "/hostkey",
		"PermitRootLogin no",
		// PAM's session stack fails in a container on pam_loginuid, and sshd
		// treats a failed session as a refused login.
		"UsePAM no",
		"PasswordAuthentication no",
		"KbdInteractiveAuthentication no",
		"PubkeyAuthentication yes",
		"AuthorizedKeysFile " + ConfDir + "/authorized_keys",
		"AllowUsers opencode",
		"AllowTcpForwarding no",
		"AllowAgentForwarding no",
		"AllowStreamLocalForwarding no",
		"GatewayPorts no",
		"X11Forwarding no",
		"PermitTunnel no",
		"PermitUserEnvironment no",
		"",
	}, "\n")
}
