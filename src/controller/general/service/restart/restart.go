package restart

import (
	"os"
	"sort"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/service"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"service:restart"},
		Handler:  Execute,
		Help:     "Restart one service of the project, leaving the others running",
		Category: "service",
		ArgsType: new(arg_struct.ControllerGeneralServiceRestart),
	})
}

// deployerService is the container a deploy runs in.
//
// Named here because it is the one service whose restart is a different kind of
// act: from inside a deploy it kills the process doing the restarting.
const deployerService = "deployer"

// Seams for the tests. Nothing else replaces them.
var (
	composeServices = docker.ComposeServices
	restartServices = docker.RestartServices
	serviceStates   = docker.ServiceStates
	projectName     = configs.GetProjectName
	exit            = os.Exit
)

// Execute restarts the named services and touches nothing else.
//
// `restart` cannot be a step of a deploy recipe: it stops every container of
// the project including `deployer`, which is the process running the recipe, so
// it dies at the moment it succeeds. Restarting after a deploy therefore stayed
// a second step for a person — and on 2026-08-19, on one machine, three of four
// projects were serving a release older than the one `current` pointed at,
// twice in the same session. Two of the three people who forgot knew about the
// trap. Something that fails that way is not fixed by remembering harder.
//
// With one service named, the recipe ends by restarting its own application and
// the deployer container lives to report the result. On a machine where several
// projects share a database it also removes the price of the blunt tool: a
// project-wide restart there takes down the database container every other
// application on the machine is connected to.
func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralServiceRestart)).(*arg_struct.ControllerGeneralServiceRestart)

	project := projectName()

	available, err := composeServices(project)
	if err != nil {
		// A question that could not be asked is its own answer, and it is not
		// "no such service". Saying which one it was is what lets somebody act.
		fmtc.ErrorLn("Could not ask docker which services this project has: " + err.Error())
		exit(1)
		return
	}

	if len(args.Args) == 0 {
		fmtc.ErrorLn("Service name is required: madock service:restart <name>")
		printAvailable(available)
		exit(1)
		return
	}

	// Every name is resolved before anything is restarted. A typo in the second
	// of two arguments would otherwise restart the first and then refuse, which
	// leaves the caller guessing which half happened.
	resolved := make([]string, 0, len(args.Args))
	for _, name := range args.Args {
		if current, renamed := service.Renamed(name); renamed {
			fmtc.WarningLn("\"" + name + "\" is now \"" + current + "\". The old name still works for now.")
		}

		svc, ok := resolve(name, available)
		if !ok {
			fmtc.ErrorLn("The service \"" + name + "\" is not part of this project's stack. Nothing was restarted.")
			printAvailable(available)
			exit(1)
			return
		}
		resolved = append(resolved, svc)
	}

	for _, svc := range resolved {
		if svc == deployerService {
			fmtc.WarningLn("\"deployer\" is the container a deploy runs in — restarting it from inside a deploy kills that deploy. That is the trap this command exists to avoid.")
		}
	}

	if err := restartServices(project, resolved); err != nil {
		fmtc.ErrorLn("Restart failed: " + err.Error())
		exit(1)
		return
	}

	report(project, resolved)
}

// report says whether the services came back, and says so from docker rather
// than from the restart having returned zero.
//
// `docker compose restart` exits zero once it has sent the signals. A container
// whose main process exits on start — a worker with a broken command, most
// often — is gone a moment later, and the state right after the restart is the
// only cheap chance to notice. It is not proof of health: a service that dies
// thirty seconds in still reports running here. It is the difference between
// "restarted" and "restarted and still there", which is the claim being made.
func report(project string, services []string) {
	states, err := serviceStates(project)
	if err != nil {
		fmtc.WarningLn("Restarted " + strings.Join(services, ", ") +
			", but docker could not be asked whether they came back: " + err.Error())
		return
	}

	state := make(map[string]string, len(states))
	for _, entry := range states {
		state[entry.Service] = entry.State
	}

	var down []string
	for _, svc := range services {
		switch current, known := state[svc]; {
		case !known:
			fmtc.WarningLn(svc + ": restarted, but docker lists no container for it")
		case current == "running" || current == "restarting":
			fmtc.Title(svc)
			fmtc.SuccessLn(" " + current)
		default:
			down = append(down, svc+" ("+current+")")
		}
	}

	if len(down) > 0 {
		fmtc.ErrorLn("Not running after the restart: " + strings.Join(down, ", "))
		exit(1)
	}
}

// resolve maps a name a person types to the service compose knows.
//
// Three spellings arrive here and all three are legitimate: the compose service
// itself (`php`, `worker-queue`, `deployer`), the short name madock prints
// elsewhere (`elasticsearch`), and the config key that switches it
// (`search/elasticsearch`). A name that matches none of them is refused rather
// than guessed at — restarting a service nobody asked for is worse than
// restarting nothing.
func resolve(name string, available []string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}

	if svc, ok := match(name, available); ok {
		return svc, true
	}
	if svc, ok := match(service.GetByLong(name), available); ok {
		return svc, true
	}
	// A name that moved — `php/nodejs` → `nodejs/embedded` → `embedded-node`.
	if current, renamed := service.Renamed(name); renamed {
		if svc, ok := match(service.GetByLong(current), available); ok {
			return svc, true
		}
	}
	return "", false
}

func match(name string, available []string) (string, bool) {
	for _, svc := range available {
		if strings.EqualFold(svc, name) {
			return svc, true
		}
	}
	return "", false
}

// printAvailable lists what the project actually has, which is the only useful
// thing to say next to "no such service".
func printAvailable(available []string) {
	if len(available) == 0 {
		fmtc.WarningLn("This project's stack declares no services.")
		return
	}
	names := append([]string(nil), available...)
	sort.Strings(names)
	fmtc.WarningLn("Services of this project: " + strings.Join(names, ", "))
}
