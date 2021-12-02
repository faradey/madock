package attr

import "strconv"

var Attributes map[string]string

func ParseAttributes(args []string) {
	Attributes = make(map[string]string)
	if len(args) > 2 {
		lastAttribute := ""
		for i, val := range args[2:] {
			if val[:2] == "--" {
				Attributes[val] = strconv.Itoa(i)
				lastAttribute = val
			} else {
				if lastAttribute == "" {
					Attributes[strconv.Itoa(i)] = val
				} else {
					Attributes[lastAttribute] = val
				}
			}
		}
	}
}
