package remove

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/ports"
)

type ArgsStruct struct {
	attr.Arguments
	Force        bool   `arg:"-f,--force" help:"Skip interactive confirmations (requires --name)"`
	Name         string `arg:"-n,--name" help:"Project name to remove (required with --force)"`
	RegistryOnly bool   `arg:"--registry-only" help:"Remove only what the installation holds for the project — registry entry, runtime, proxy block, ports and containers. The project directory is never touched (requires --name)"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"project:remove"},
		Handler:  Execute,
		Help:     "Remove project",
		Category: "project",
		ArgsType: new(ArgsStruct),
		// Runs outside a project on purpose, and only for --name.
		//
		// The scope check refuses a project command run anywhere else, which is
		// right for every other one and wrong for this: an orphan is an entry
		// whose source directory is gone, so there is nowhere to stand to remove
		// it. project:list names such entries and nothing could remove them —
		// the tool could describe the problem and not act on it.
		//
		// Without --name the handler still requires a project directory, so the
		// protection that refuses to destroy the wrong one is unchanged.
		Global: true,
	})
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	// Before anything is read, decided or printed: this command deletes a
	// project's data, and an installation may forbid that.
	//
	// --registry-only answers this for itself, once it knows which entry was
	// named. It has to: what such a removal costs is a property of the entry,
	// not of the command, and that cannot be known here.
	if !args.RegistryOnly && !configs.AllowsDestructiveCommands() {
		for _, line := range configs.DestructiveRefusal("project:remove") {
			fmtc.ErrorLn(line)
		}
		os.Exit(1)
	}

	// Asked for explicitly, and answered before anything looks at the working
	// directory: this is the one form that is defined not to touch the source.
	//
	// A flag rather than a rule inside the existing branch. A destructive command
	// should do what its invocation says and nothing else — an entry that quietly
	// removes less because the code guessed the caller meant less is how somebody
	// learns, once, that it can also guess the other way.
	if args.RegistryOnly {
		removeRegistryOnly(args.Name, args.Force)
		return
	}

	// A name is answered by the registry, not by the working directory.
	if args.Name != "" && args.Name != configs.GetProjectName() {
		removeNamed(args.Name, args.Force)
		return
	}

	projectName := configs.GetProjectName()

	if !mayRemove(projectName) {
		os.Exit(1)
	}

	// Non-interactive mode with --force flag
	if args.Force {
		if args.Name == "" {
			fmtc.ErrorLn("--force requires --name to specify the project name")
			return
		}
		if args.Name != projectName {
			fmtc.ErrorLn("Project name mismatch. Current project: " + projectName + ", specified: " + args.Name)
			return
		}
		removeProject(projectName)
		return
	}

	// Interactive mode
	fmt.Println("Are you sure? (y/n)")
	fmt.Print("> ")
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		logger.Fatalln(err)
	}
	result := strings.ToLower(strings.TrimSpace(string(sentence)))
	if result == "y" && len(projectName) > 0 {
		pp := paths.NewProjectPaths(projectName)
		fmt.Println("The following items will be removed:")
		fmt.Println(paths.GetExecDirPath() + "/projects/" + projectName + "/")
		fmt.Println(pp.RuntimeDir())
		for _, line := range removalTargetLines(paths.GetRunDirPath()) {
			fmt.Println(line)
		}
		fmt.Println("Containers, images and volumes associated with the project.")
		fmt.Println("")
		fmt.Println("Enter the project name \"" + projectName + "\" to confirm the deletion of the project")
		fmt.Print("> ")
		buf = bufio.NewReader(os.Stdin)
		sentence, err = buf.ReadBytes('\n')
		if err != nil {
			logger.Fatalln(err)
		}
		result = strings.TrimSpace(string(sentence))
		if result == projectName {
			removeProject(projectName)
		} else {
			fmtc.WarningLn("The project was not removed. The entered value does not match the project name.")
		}
	}
}

// mayRemove answers whether this directory may be destroyed at all.
//
// removeProject finishes with RemoveAll on the current directory, and until now
// the only thing standing between that and the wrong directory was the name: the
// project name comes from the directory name, and --force merely checks that the
// caller repeated it. A leftover runtime directory with no configuration was
// enough to make any same-named directory look like a project — and one such
// leftover was the madock installation itself, whose runtime `src` was a symlink
// back to the source tree. `project:remove --force --name madock` there would have
// deleted madock, its repository and every other project's configuration with it.
//
// So three refusals before anything is touched: the installation, a directory that
// is not the project's own, and a project that does not exist.
func mayRemove(projectName string) bool {
	runDir := paths.GetRunDirPath()

	if configs.IsSamePath(runDir, paths.GetExecDirPath()) {
		fmtc.ErrorLn("This directory is the madock installation, not a project — refusing to remove it")
		return false
	}

	if !paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml") {
		fmtc.ErrorLn("There is no project '" + projectName + "' to remove: it has no configuration")
		fmtc.ToDoLn("Leftover generated files, if any, are under " + paths.GetExecDirPath() + "/aruntime/projects/" + projectName)
		return false
	}

	// A recorded path that disagrees with where we stand means the name resolved
	// to somebody else's project, and RemoveAll would take this directory apart on
	// its behalf.
	stored := configs.GetProjectConfigOnly(projectName)["path"]
	if stored != "" && !configs.IsSamePath(stored, runDir) {
		fmtc.ErrorLn("The project '" + projectName + "' is registered at another path, so this directory is not it")
		fmtc.ErrorLn("  registered: " + stored)
		fmtc.ErrorLn("  current:    " + runDir)
		return false
	}

	return true
}

// resolvedRemovalPath answers what os.RemoveAll(dir) will actually destroy.
//
// The scale of this command is a property of the shell, not of the command. The
// directory it deletes is os.Getwd(), which keeps whatever symlinks the shell
// walked through, so one line typed two ways destroys two different things. On a
// Deployer layout — releases/N with a `current` symlink pointing at one of them —
// this was measured rather than reasoned about:
//
//	cd .../current      Getwd keeps the link  → RemoveAll unlinks `current`, releases survive
//	cd -P .../current   Getwd is physical     → RemoveAll takes the whole release directory
//
// Both printed the same warning, naming `.../current`, which is true of neither
// half of that table. So the second return says whether the path is itself a
// link (the link goes, the target stays) and the first names the directory that
// really goes when it is not.
func resolvedRemovalPath(dir string) (string, bool) {
	if dir == "" {
		return dir, false
	}

	if info, err := os.Lstat(dir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if target, err := filepath.EvalSymlinks(dir); err == nil {
			return target, true
		}
		if target, err := os.Readlink(dir); err == nil {
			return target, true
		}
		return dir, true
	}

	// Not a link itself, but a parent component may be one — and then RemoveAll
	// reaches straight through it into the real directory.
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		return resolved, false
	}

	return dir, false
}

// removalTargetLines is what the person confirming gets to read.
//
// It says the resolved path out loud whenever it differs from the one typed:
// in a release directory that is the difference between `.../current` and
// `.../releases/847`, and nobody stops at the first spelling.
func removalTargetLines(dir string) []string {
	target, isLink := resolvedRemovalPath(dir)

	switch {
	case isLink && target != dir:
		return []string{
			dir,
			"    a symlink: the link goes, " + target + " stays",
		}
	case !isLink && target != dir:
		return []string{
			dir,
			"    resolves to " + target + " — that is the directory that will be deleted",
		}
	}

	return []string{dir}
}

func removeProject(projectName string) {
	// Say what is about to go even when --force skipped the interactive listing:
	// this is the one command that deletes the directory the caller is standing in.
	pp := paths.NewProjectPaths(projectName)
	fmtc.WarningLn("Removing project '" + projectName + "':")
	fmtc.WarningLn("  " + paths.GetExecDirPath() + "/projects/" + projectName + "/")
	fmtc.WarningLn("  " + pp.RuntimeDir())
	for _, line := range removalTargetLines(paths.GetRunDirPath()) {
		fmtc.WarningLn("  " + line)
	}
	fmtc.WarningLn("  containers, images and volumes of the project")

	removeRegistered(projectName, true, true)

	// The project's own directory, and only when standing in it. Split out so
	// that removing an entry by name — where the source is gone and the caller
	// is standing somewhere else entirely — cannot reach this line.
	if err := os.RemoveAll(paths.GetRunDirPath()); err != nil {
		logger.Fatal(err)
	}
}

// removeRegistered takes down everything the installation holds for a project:
// its containers, its registry entry, its generated runtime, its block in the
// shared proxy and its port reservation. Everything except the project's own
// directory, which is the caller's to decide about.
func removeRegistered(projectName string, reclaimSource, withVolumes bool) {
	pp := paths.NewProjectPaths(projectName)

	// Before the containers go: anything they wrote as root has to be handed
	// back, or the deletion below stops at the first such file and leaves the
	// project half removed.
	//
	// Only when the source directory is this command's to delete. ReclaimProjectFiles
	// falls back to a container that mounts the *working directory* and chowns it
	// recursively — harmless when that directory is the project being destroyed,
	// and not harmless at all when the caller is standing somewhere else, which is
	// every call that removes an entry by name.
	if reclaimSource {
		docker.ReclaimProjectFiles(projectName)
	}

	// withVolumes false is what makes a registry-only removal safe to allow on an
	// installation that forbids destructive commands: containers are not data and
	// come back from the configuration, volumes and images are and do not.
	docker.Down(projectName, withVolumes)

	if err := os.RemoveAll(paths.GetExecDirPath() + "/projects/" + projectName + "/"); err != nil {
		logger.Fatal(err)
	}

	if err := os.RemoveAll(pp.RuntimeDir()); err != nil {
		logger.Fatal(err)
	}

	// Take the project out of the shared proxy.
	//
	// Removal is the one moment this belongs to. A stopped project keeps its
	// routing on purpose — it is coming back, and rewriting every other
	// project's configuration for a pause would be churn and risk for nothing.
	// A removed project is not coming back, and its server block points at a
	// container that no longer exists.
	//
	// Nothing did this before, so the block survived until something else
	// happened to regenerate the file.
	// Regenerated on behalf of a project that still exists, never the removed
	// one: MakeConf recreates directories and allocates ports for the name it is
	// given, so naming the corpse brings its registry entry and its port
	// reservation straight back.
	remaining := paths.GetActiveProjects()
	if len(remaining) > 0 {
		nginx.MakeConf(remaining[0])
		if err := docker.ReloadNginx(); err != nil {
			logger.Println(err)
		}
	}

	// Ports come last, and the order is not cosmetic: generating the proxy
	// configuration allocates a port for the project it is given, so releasing
	// them first only meant handing them straight back. The removed project then
	// kept a reservation no other project could use.
	ports.GetRegistry().RemoveProject(projectName)

	fmtc.SuccessLn("Project was removed successfully")
	fmtc.SuccessLn("!!! Close the terminal for the changes to take effect !!!")
}

// removeNamed removes a registry entry from anywhere, by name.
//
// It exists for the orphan: an entry whose source directory is gone. project:list
// has been able to name those since 3.8.50 and nothing could remove them, because
// this command asked the working directory who the project was — and for an
// orphan there is no directory to stand in. The workaround was to recreate the
// directory with a copy of its config.xml and delete from there, which is a
// strange thing for a tool to require of the person cleaning up after it.
//
// A project whose source still exists is refused and told where it is. That keeps
// the protection this command is built around: removeProject ends with RemoveAll
// on the working directory, so removing a live project from somewhere else would
// take that somewhere else with it.
func removeNamed(projectName string, force bool) {
	var entry *configs.ProjectEntry
	for _, candidate := range configs.ListProjects() {
		if candidate.Name == projectName {
			found := candidate
			entry = &found
			break
		}
	}

	if entry == nil {
		fmtc.ErrorLn("No project named '" + projectName + "' is registered in this installation")
		fmtc.ToDoLn("madock project:list")
		os.Exit(1)
	}

	// Registered inside somebody else's project. The advice below must not be
	// printed for this one: the directory is another project's live release, and
	// "cd there and remove it" ends in a recursive delete of it.
	if entry.State == configs.ProjectNestedPath {
		fmtc.ErrorLn("Entry '" + projectName + "' is registered inside another project: " + entry.Path)
		if entry.Owner != "" {
			fmtc.ErrorLn("  that directory belongs to '" + entry.Owner + "'")
		}
		fmtc.ErrorLn("Do not remove it from there: this command deletes the directory it is run in.")
		fmtc.ToDoLn("Drop the installation's record of it and leave the directory alone:")
		fmtc.ToDoLn("  madock project:remove --name=" + projectName + " --registry-only")
		os.Exit(1)
	}

	if entry.State == configs.ProjectOk {
		fmtc.ErrorLn("Project '" + projectName + "' still has its source directory: " + entry.Path)
		fmtc.ToDoLn("Remove it from there, so the directory goes with it:")
		fmtc.ToDoLn("  cd " + entry.Path + " && madock project:remove --force --name=" + projectName)
		fmtc.ToDoLn("Or drop only the installation's record of it, keeping the directory:")
		fmtc.ToDoLn("  madock project:remove --name=" + projectName + " --registry-only")
		os.Exit(1)
	}

	fmtc.WarningLn("Removing the registry entry '" + projectName + "':")
	fmtc.WarningLn("  " + paths.GetExecDirPath() + "/projects/" + projectName + "/")
	fmtc.WarningLn("  " + paths.NewProjectPaths(projectName).RuntimeDir())
	fmtc.WarningLn("  containers, images and volumes of the project")
	if entry.Path != "" {
		fmtc.WarningLn("  its source directory is already gone: " + entry.Path)
	}

	if !force {
		fmt.Println("")
		fmt.Println("Enter the project name \"" + projectName + "\" to confirm")
		fmt.Print("> ")
		buf := bufio.NewReader(os.Stdin)
		sentence, err := buf.ReadBytes('\n')
		if err != nil {
			logger.Fatalln(err)
		}
		if strings.TrimSpace(string(sentence)) != projectName {
			fmtc.WarningLn("The project was not removed. The entered value does not match the project name.")
			return
		}
	}

	// No reclaim: the source directory is gone, so there is nothing to hand back
	// — and the caller is standing somewhere unrelated, which is the one place
	// the fallback would chown. Volumes do go: this branch runs only for an entry
	// whose source is gone, and it is gated on the installation allowing it.
	removeRegistered(projectName, false, true)
}

// removeRegistryOnly drops what the installation holds for a name and nothing else.
//
// The case it is written for is an entry whose directory exists and is not its
// own: madock run inside a Deployer release registers a project called `current`,
// pointing into a live application. Three of those were found on one server. The
// entry is not harmless — it holds ports and a server block in the shared proxy —
// and until now there was no way to remove it: removeNamed refuses while the
// directory exists, and its advice was to go and remove it from there, which ends
// with a recursive delete of that application's release.
//
// So the source is not merely left alone here, it is unreachable from this path:
// nothing below takes a directory, and removeRegistered has never deleted one.
// registryOnlyAllowed decides whether a registry-only removal may run on an
// installation that forbids destructive commands.
//
// The ban exists to stop a project's data being destroyed, and for a healthy
// project this removal still does that: the entry it deletes is the project's
// madock configuration — database passwords, ports, the stack — and nothing
// recreates it. So the ban holds there.
//
// It does not hold for an entry that is not a project of its own. A record whose
// source directory is gone, whose link resolves to nothing, or whose path lives
// inside another project owns nothing that could be lost: what it holds is a
// port reservation and a block in the shared proxy, both on behalf of something
// that does not exist. Refusing there protected nothing and left three such
// entries on a production host with no way to remove them — the machine that
// most needs the ban is the machine where they accumulate.
//
// Paired with withVolumes=false in this path, so what runs under the exemption
// cannot delete data even if an entry turns out to have some.
func registryOnlyAllowed(state string, destructiveAllowed bool) bool {
	if destructiveAllowed {
		return true
	}

	switch state {
	case configs.ProjectMissingSource, configs.ProjectBrokenLink, configs.ProjectNestedPath:
		return true
	}

	// ProjectOk, and ProjectNoPath — a legacy entry of a project that may well
	// still exist, which is not something to guess about under a ban.
	return false
}

func removeRegistryOnly(projectName string, force bool) {
	// Taken from --name only. The working directory is what invented these
	// entries in the first place, and a command whose entire promise is "the
	// directory is not the target" must not be reading the target off it.
	if projectName == "" {
		fmtc.ErrorLn("--registry-only requires --name: the entry to drop is named, never taken from the working directory")
		fmtc.ToDoLn("madock project:list --stale")
		os.Exit(1)
	}

	var entry *configs.ProjectEntry
	for _, candidate := range configs.ListProjects() {
		if candidate.Name == projectName {
			found := candidate
			entry = &found
			break
		}
	}

	if entry == nil {
		fmtc.ErrorLn("No project named '" + projectName + "' is registered in this installation")
		fmtc.ToDoLn("madock project:list")
		os.Exit(1)
	}

	if !registryOnlyAllowed(entry.State, configs.AllowsDestructiveCommands()) {
		for _, line := range configs.DestructiveRefusal("project:remove --registry-only") {
			fmtc.ErrorLn(line)
		}
		fmtc.ErrorLn("")
		fmtc.ErrorLn("'" + projectName + "' is a project in its own right, and its record is its madock")
		fmtc.ErrorLn("configuration — passwords, ports, the stack. Nothing recreates that.")
		fmtc.ToDoLn("Entries that are not a project of their own are removable here without changing")
		fmtc.ToDoLn("that setting — source gone, link broken, or path inside another project:")
		fmtc.ToDoLn("  madock project:list --stale")
		os.Exit(1)
	}

	fmtc.WarningLn("Removing the installation's record of '" + projectName + "':")
	fmtc.WarningLn("  " + paths.GetExecDirPath() + "/projects/" + projectName + "/")
	fmtc.WarningLn("  " + paths.NewProjectPaths(projectName).RuntimeDir())
	fmtc.WarningLn("  its block in the shared proxy and its port reservation")
	fmtc.WarningLn("  containers, images and volumes of the project")
	if entry.Path != "" {
		fmtc.SuccessLn("  the directory itself is left alone: " + entry.Path)
	}
	if entry.State == configs.ProjectNestedPath && entry.Owner != "" {
		fmtc.SuccessLn("  it belongs to project '" + entry.Owner + "', which keeps its own entry")
	}

	if !force {
		fmt.Println("")
		fmt.Println("Enter the project name \"" + projectName + "\" to confirm")
		fmt.Print("> ")
		buf := bufio.NewReader(os.Stdin)
		sentence, err := buf.ReadBytes('\n')
		if err != nil {
			logger.Fatalln(err)
		}
		if strings.TrimSpace(string(sentence)) != projectName {
			fmtc.WarningLn("Nothing was removed. The entered value does not match the project name.")
			return
		}
	}

	// Neither the source nor the project's data: containers go, volumes and images
	// stay. An orphaned volume is recoverable and findable — `prune` and the
	// orphans command exist for exactly that — while a volume deleted under an
	// installation that forbids destructive commands is the thing the setting was
	// put there to prevent.
	removeRegistered(projectName, false, false)
}

// definition returns this command's registration, for tests that assert on it.
func definition() *command.Definition {
	def, _ := command.Get("project:remove")
	return def
}
