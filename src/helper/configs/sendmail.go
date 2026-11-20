package configs

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// systemSendmail is what PHP uses when sendmail_path is not set, spelled out:
// the php.ini line now always reads the environment, so "off" has to name the
// default rather than leave the line commented out.
const systemSendmail = "/usr/sbin/sendmail -t -i"

var (
	sendmailHost = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)
	sendmailFrom = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$`)
)

// SendmailPath is PHP's sendmail_path for the project, the value every PHP
// service gets as MADOCK_SENDMAIL_PATH. It is computed from the configuration
// already passed through the derived rules, so a muted clone's or an
// extension's host and sender are the ones in it.
//
// A configuration ValidateSendmail refuses yields "": the image is never built
// from one, and an empty sendmail_path makes mail() fail rather than run a
// command nobody checked.
func SendmailPath(conf map[string]string) string {
	if strings.TrimSpace(conf["php/sendmail/enabled"]) != "true" {
		return systemSendmail
	}
	if ValidateSendmail(conf) != nil {
		return ""
	}

	path := "/usr/bin/msmtp -t --port=" + strings.TrimSpace(conf["php/sendmail/port"]) +
		" --host=" + strings.TrimSpace(conf["php/sendmail/host"])
	if from := strings.TrimSpace(conf["php/sendmail/from"]); from != "" {
		path += " --from=" + from
	}

	return path
}

// ValidateSendmail reports php/sendmail values that cannot become a working
// sendmail_path.
//
// The three values are joined into a command that PHP hands to /bin/sh on every
// mail() call, and the command travels through a compose file on the way, where
// a "$" is interpolation and a quote ends the string. So they are checked
// against what each of them is — a host name, a port number, an address — and
// not escaped: an address with a space or a semicolon in it is not one msmtp
// would accept anyway, and refusing it here says so before an image is built
// instead of in the mail log of a running application.
func ValidateSendmail(conf map[string]string) error {
	if strings.TrimSpace(conf["php/sendmail/enabled"]) != "true" {
		return nil
	}

	host := strings.TrimSpace(conf["php/sendmail/host"])
	if !sendmailHost.MatchString(host) {
		return fmt.Errorf("php/sendmail/host is %q, which is not a host name. mail() would run it as part of a shell command", host)
	}

	port := strings.TrimSpace(conf["php/sendmail/port"])
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("php/sendmail/port is %q, which is not a port number", port)
	}

	if from := strings.TrimSpace(conf["php/sendmail/from"]); from != "" && !sendmailFrom.MatchString(from) {
		return fmt.Errorf("php/sendmail/from is %q, which is not a plain address (name@domain). "+
			"msmtp takes the envelope sender alone, without a display name", from)
	}

	return nil
}
