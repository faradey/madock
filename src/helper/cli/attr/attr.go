package attr

var IsParseArgs = true

type Arguments struct {
}

type ArgumentsWithArgs struct {
	Arguments
	Args []string `arg:"positional"`
}
