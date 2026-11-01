package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

const compose = `name: madock_x
services:
  php:
    init: true
    build:
      context: ctx
      dockerfile: php.Dockerfile
    volumes:
      - ./src:/var/www/html:cached
      - ./composer:/var/www/.composer:cached
      - ./ssh:/var/www/.ssh:ro
    environment:
      - COMPOSER_HOME=/var/www/.composer
    extra_hosts:
      - "host.docker.internal:host-gateway"
    restart: no
  db:
    image: mariadb

volumes:
  dbdata:

networks:
  madock-proxy:
    external: true
`

func service(t *testing.T, content, name string) string {
	t.Helper()
	block := serviceBlock(content, name)
	if block == nil {
		t.Fatalf("no service %s in:\n%s", name, content)
	}
	return strings.Join(block, "\n")
}

// The point of the sandbox: opencode itself is on the internal network only.
// One extra network here and a shell command reaches the database.
func TestOpencodeIsOnTheInternalNetworkOnly(t *testing.T) {
	out := InjectCompose(compose, "php")
	got := strings.Join(section(serviceBlock(out, Service), "networks"), "\n")
	if got != "    networks:\n      - opencode" {
		t.Fatalf("opencode networks:\n%s", got)
	}
	if !strings.Contains(out, "  opencode:\n    driver: bridge\n    internal: true\n") {
		t.Fatalf("the opencode network is not internal:\n%s", out)
	}
}

// opencode-exec gets the main service's build and code, and nothing that
// carries the host user's keys or tokens.
func TestExecTakesAnAllowlistFromTheMainService(t *testing.T) {
	exec := service(t, InjectCompose(compose, "php"), ExecService)
	for _, want := range []string{"dockerfile: php.Dockerfile", "- ./src:/var/www/html:cached", "host.docker.internal:host-gateway", "user: root"} {
		if !strings.Contains(exec, want) {
			t.Errorf("opencode-exec lacks %q:\n%s", want, exec)
		}
	}
	for _, leak := range []string{"./ssh:", "./composer:", "COMPOSER_HOME"} {
		if strings.Contains(exec, leak) {
			t.Errorf("opencode-exec carries %q from the main service:\n%s", leak, exec)
		}
	}
}

// A main service without a networks key is on the default network. Naming
// only the opencode network would take opencode-exec off it, and bin/magento
// would lose the database.
func TestExecKeepsTheDefaultNetwork(t *testing.T) {
	exec := service(t, InjectCompose(compose, "php"), ExecService)
	if !strings.Contains(exec, "    networks:\n      - default\n      - opencode") {
		t.Fatalf("opencode-exec networks:\n%s", exec)
	}
}

func TestExecTakesTheMainServiceNetworks(t *testing.T) {
	isolated := strings.Replace(compose, "    restart: no\n  db:", "    networks:\n      - isolated\n    restart: no\n  db:", 1)
	exec := service(t, InjectCompose(isolated, "php"), ExecService)
	if !strings.Contains(exec, "    networks:\n      - isolated\n      - opencode") {
		t.Fatalf("opencode-exec networks:\n%s", exec)
	}
}

func TestComposeWithoutTheMainServiceIsLeftAlone(t *testing.T) {
	if out := InjectCompose(compose, "python"); out != compose {
		t.Fatalf("changed a file with no python service:\n%s", out)
	}
}

func TestInjectComposeIsIdempotent(t *testing.T) {
	once := InjectCompose(compose, "php")
	if twice := InjectCompose(once, "php"); twice != once {
		t.Fatal("a second pass changed the file")
	}
}

func TestDockerfileGivesTheImageItsUserBack(t *testing.T) {
	out := InjectDockerfile("FROM ruby\nUSER ruby\nCMD [\"rails\"]\n")
	if !strings.Contains(out, "# "+marker+" (sshd runs only in the opencode-exec service)\nUSER root\nRUN apt-get") {
		t.Fatalf("the block does not switch to root:\n%s", out)
	}
	if !strings.HasSuffix(out, "MADOCK_EOF\nUSER ruby\n") {
		t.Fatalf("the image does not end as ruby again:\n%s", out)
	}
	if plain := InjectDockerfile("FROM php\n"); strings.Contains(plain, "\nUSER ") {
		t.Fatalf("an image with no USER got one:\n%s", plain)
	}
}

func TestOnlyTheMainImageGetsSshd(t *testing.T) {
	if !MainDockerfile("php", "php.DockerfileWithoutXdebug") || MainDockerfile("php", "nodejs.Dockerfile") {
		t.Fatal("a php project")
	}
	if !MainDockerfile("app", "app.Dockerfile") || MainDockerfile("app", "php.Dockerfile") {
		t.Fatal("a language-less project")
	}
}

func TestUpstreamSplitsOriginFromPath(t *testing.T) {
	origin, path, err := Upstream("http://host.docker.internal:8080/v1/")
	if err != nil || origin != "http://host.docker.internal:8080" || path != "/v1" {
		t.Fatalf("got %q %q %v", origin, path, err)
	}
	for _, bad := range []string{"", "host.docker.internal:8080", "ftp://x/v1", "http:///v1"} {
		if _, _, err := Upstream(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestLLMConfForwardsToOneUpstream(t *testing.T) {
	conf, err := LLMConf("https://llm.example.com/api/v1")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"proxy_pass https://llm.example.com;", "proxy_set_header Host llm.example.com;", "proxy_ssl_server_name on;"} {
		if !strings.Contains(conf, want) {
			t.Errorf("missing %q:\n%s", want, conf)
		}
	}
	if strings.Count(conf, "proxy_pass") != 1 {
		t.Fatalf("more than one upstream:\n%s", conf)
	}
}

func conf() map[string]string {
	return map[string]string{
		"opencode/base_url": "http://host.docker.internal:8080/v1",
		"opencode/model":    "gpt-oss-20b-64k",
		"opencode/context":  "65536",
		"opencode/output":   "8192",
	}
}

func TestConfigPointsAtTheProxy(t *testing.T) {
	raw, err := Config(conf())
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Provider map[string]struct {
			Options map[string]string
			Models  map[string]struct{ Limit map[string]int }
		}
		Model      string
		Permission map[string]string
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	p := doc.Provider["madock"]
	if p.Options["baseURL"] != "http://opencode-llm:8080/v1" {
		t.Errorf("baseURL %q", p.Options["baseURL"])
	}
	if doc.Model != "madock/gpt-oss-20b-64k" || p.Models["gpt-oss-20b-64k"].Limit["context"] != 65536 {
		t.Errorf("model %q, limits %v", doc.Model, p.Models)
	}
	if doc.Permission["external_directory"] != "deny" {
		t.Errorf("permission %v", doc.Permission)
	}
}

func TestConfigRefusesNonsense(t *testing.T) {
	for key, value := range map[string]string{"opencode/model": " ", "opencode/context": "64k", "opencode/output": "0", "opencode/base_url": "localhost:8080"} {
		c := conf()
		c[key] = value
		if _, err := Config(c); err == nil {
			t.Errorf("%s=%q accepted", key, value)
		}
	}
}

func TestAgentsFileIsReadOrRefused(t *testing.T) {
	if got, _ := Agents(map[string]string{}); got != DefaultAgents {
		t.Fatal("no file named, but not the default")
	}
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	if err := os.WriteFile(path, []byte("own rules\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := Agents(map[string]string{"opencode/agents_file": path}); err != nil || got != "own rules\n" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := Agents(map[string]string{"opencode/agents_file": path + ".missing"}); err == nil {
		t.Fatal("a missing file fell back to the default")
	}
}

// The keys survive a second run: a host key that changed on every rebuild is
// one the strict client refuses.
func TestKeysAreGeneratedOnceAndAgree(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureKeys(dir); err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte {
		raw, err := os.ReadFile(filepath.Join(dir, keysDir, name))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	host, client := read(hostKeyFile), read(clientKeyFile)
	if err := EnsureKeys(dir); err != nil {
		t.Fatal(err)
	}
	if string(read(hostKeyFile)) != string(host) || string(read(clientKeyFile)) != string(client) {
		t.Fatal("a second run replaced a key")
	}
	info, err := os.Stat(filepath.Join(dir, keysDir, clientKeyFile))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("client key mode %v %v", info.Mode(), err)
	}

	signer, err := ssh.ParsePrivateKey(client)
	if err != nil {
		t.Fatal(err)
	}
	authorized, _, _, _, err := ssh.ParseAuthorizedKey(read(authorizedKeysFile))
	if err != nil || string(authorized.Marshal()) != string(signer.PublicKey().Marshal()) {
		t.Fatal("authorized_keys does not hold the client key")
	}
	hostSigner, err := ssh.ParsePrivateKey(host)
	if err != nil {
		t.Fatal(err)
	}
	_, hosts, known, _, _, err := ssh.ParseKnownHosts(read(knownHostsFile))
	if err != nil || len(hosts) != 1 || hosts[0] != ExecService || string(known.Marshal()) != string(hostSigner.PublicKey().Marshal()) {
		t.Fatalf("known_hosts: %v %v", hosts, err)
	}
}
