package main

import "strings"

var passwordManagerNames = []string{
	"keepass", "keepassxc", "lastpass", "1password", "bitwarden",
	"dashlane", "nordpass", "roboform", "keeper", "enpass",
	"gopass", "pass", "password manager", "credential",
}

func isPasswordManagerName(name string) bool {
	lower := strings.ToLower(name)
	for _, pm := range passwordManagerNames {
		if strings.Contains(lower, pm) {
			return true
		}
	}
	return false
}
