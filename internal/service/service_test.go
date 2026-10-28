package service

import (
	"testing"
)

func TestParseName(t *testing.T) {
	input := []struct {
		input     string
		wantOwner string
		wantName  string
		wantErr   bool
	}{
		{"golang/go", "golang", "go", false},
		{"golang", "", "", true},
		{"", "", "", true},
		{"golang/go/golang", "", "", true},
	}

	for _, test := range input {
		owner, name, err := parseName(test.input)
		if err != nil && !test.wantErr {
			t.Errorf("parseName(%q): unexpected error: %v", test.input, err)
		}
		if err == nil && test.wantErr {
			t.Errorf("parseName() error = %v, wantErr %v", err, test.wantErr)
		}
		if owner != test.wantOwner {
			t.Errorf("parseName() error = %v, owner = %v, want %v", err, owner, test.wantOwner)
		}
		if name != test.wantName {
			t.Errorf("parseName() error = %v, name %v, want %v", err, name, test.wantName)
		}
	}
}
