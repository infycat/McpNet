package main

import "testing"

func TestAffirmativeDialogAnswer(t *testing.T) {
	tests := []struct {
		answer string
		want   bool
	}{
		{answer: "Yes", want: true},
		{answer: "yes", want: true},
		{answer: "是(Y)", want: true},
		{answer: "是(&Y)", want: true},
		{answer: "允許 / Allow", want: true},
		{answer: "No", want: false},
		{answer: "否(N)", want: false},
		{answer: "Cancel", want: false},
		{answer: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.answer, func(t *testing.T) {
			if got := affirmativeDialogAnswer(tt.answer); got != tt.want {
				t.Fatalf("affirmativeDialogAnswer(%q) = %v, want %v", tt.answer, got, tt.want)
			}
		})
	}
}
