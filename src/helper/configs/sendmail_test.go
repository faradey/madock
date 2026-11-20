package configs

import (
	"strings"
	"testing"
)

func mailConf(host, port, from string) map[string]string {
	return map[string]string{
		"php/sendmail/enabled": "true",
		"php/sendmail/host":    host,
		"php/sendmail/port":    port,
		"php/sendmail/from":    from,
	}
}

func TestSendmailPathIsTheSystemSendmailWhenMailIsOff(t *testing.T) {
	conf := mailConf("$(id)", "x", "")
	conf["php/sendmail/enabled"] = "false"

	if got := SendmailPath(conf); got != "/usr/sbin/sendmail -t -i" {
		t.Errorf("SendmailPath = %q, want PHP's own default spelled out", got)
	}
	if err := ValidateSendmail(conf); err != nil {
		t.Errorf("values that are not used were refused: %v", err)
	}
}

func TestSendmailPathNamesHostPortAndSender(t *testing.T) {
	cases := []struct {
		conf map[string]string
		want string
	}{
		{mailConf("host.docker.internal", "1025", ""), "/usr/bin/msmtp -t --port=1025 --host=host.docker.internal"},
		{mailConf(" egress ", "1025", "noreply@shop.example.com"), "/usr/bin/msmtp -t --port=1025 --host=egress --from=noreply@shop.example.com"},
		{mailConf("madock_shop-egress", "587", "a.b+c@example.com"), "/usr/bin/msmtp -t --port=587 --host=madock_shop-egress --from=a.b+c@example.com"},
	}
	for _, c := range cases {
		if got := SendmailPath(c.conf); got != c.want {
			t.Errorf("SendmailPath(%v) = %q, want %q", c.conf, got, c.want)
		}
	}
}

// Every value here would reach /bin/sh through mail(), or break the quoted
// string in the compose file on the way. None of them may produce a command.
func TestSendmailRefusesWhatIsNotAHostAPortOrAnAddress(t *testing.T) {
	cases := map[string]map[string]string{
		"host with a command":    mailConf("relay;id", "25", ""),
		"host with substitution": mailConf("$(id)", "25", ""),
		"host with a space":      mailConf("relay --host=evil", "25", ""),
		"host with a quote":      mailConf(`relay"`, "25", ""),
		"empty host":             mailConf("", "25", ""),
		"port not a number":      mailConf("relay", "25 --host=evil", ""),
		"port zero":              mailConf("relay", "0", ""),
		"port too large":         mailConf("relay", "70000", ""),
		"empty port":             mailConf("relay", "", ""),
		"sender with a name":     mailConf("relay", "25", "Shop <shop@example.com>"),
		"sender with a command":  mailConf("relay", "25", "a@b.example;id"),
		"sender with a dollar":   mailConf("relay", "25", "$USER@example.com"),
		"sender with a newline":  mailConf("relay", "25", "a@example.com\nauto_prepend_file=/etc/passwd"),
		"sender without domain":  mailConf("relay", "25", "shop@"),
	}
	for name, conf := range cases {
		if err := ValidateSendmail(conf); err == nil {
			t.Errorf("%s: accepted %v", name, conf)
		}
		if got := SendmailPath(conf); got != "" {
			t.Errorf("%s: SendmailPath = %q, want empty — a refused value must not become a command", name, got)
		}
	}
}

func TestSendmailRefusalNamesTheKey(t *testing.T) {
	err := ValidateSendmail(mailConf("relay", "25", "Shop <shop@example.com>"))
	if err == nil || !strings.Contains(err.Error(), "php/sendmail/from") {
		t.Errorf("error = %v, want it to name php/sendmail/from", err)
	}
}

// The path is built after the muted-clone rule and after extensions, so it is
// the clone's mail address and not the source's relay that ends up in it.
func TestMutedCloneSendmailPathIsTheMutedOne(t *testing.T) {
	conf := map[string]string{
		"clone/muted":          "true",
		"php/sendmail/enabled": "false",
		"php/sendmail/host":    "relay.example.com",
		"php/sendmail/port":    "25",
	}
	applyDerived(conf)

	if want := "/usr/bin/msmtp -t --port=1025 --host=host.docker.internal"; conf["php/sendmail/path"] != want {
		t.Errorf("php/sendmail/path = %q, want %q", conf["php/sendmail/path"], want)
	}
}
