package main

import "testing"

func TestTaskIDFromArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want uint
	}{
		{"notification click", []string{"taskmax://task/12"}, 12},
		{"trailing slash", []string{"taskmax://task/7/"}, 7},
		{"among other args", []string{"--flag", "taskmax://task/3"}, 3},
		{"normal launch", []string{}, 0},
		{"unrelated args", []string{"--help"}, 0},
		{"garbage id", []string{"taskmax://task/abc"}, 0},
		{"empty id", []string{"taskmax://task/"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := taskIDFromArgs(tc.args); got != tc.want {
				t.Errorf("taskIDFromArgs(%v) = %d, want %d", tc.args, got, tc.want)
			}
		})
	}
}
