package main

import "strings"

func affirmativeDialogAnswer(answer string) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "yes", "是", "是(y)", "是(&y)", "允許 / allow":
		return true
	default:
		return false
	}
}
