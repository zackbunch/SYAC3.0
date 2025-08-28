package version

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      Version
		expectErr bool
	}{
		{
			name:  "Valid version",
			input: "1.2.3",
			want:  Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:      "Missing patch",
			input:     "1.2",
			expectErr: true,
		},
		{
			name:      "Too many segments",
			input:     "1.2.3.4",
			expectErr: true,
		},
		{
			name:      "Non-numeric parts",
			input:     "a.b.c",
			expectErr: true,
		},
		{
			name:      "Invalid major version",
			input:     "a.2.3",
			expectErr: true,
		},
		{
			name:      "Invalid minor version",
			input:     "1.b.3",
			expectErr: true,
		},
		{
			name:      "Invalid patch version",
			input:     "1.2.c",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none for input %q", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("expected %+v, got %+v", tt.want, got)
				}
			}
		})
	}
}

func TestInc(t *testing.T) {
	base := Version{Major: 1, Minor: 2, Patch: 3}

	tests := []struct {
		name string
		bump VersionType
		want Version
	}{
		{
			name: "Patch bump",
			bump: Patch,
			want: Version{Major: 1, Minor: 2, Patch: 4},
		},
		{
			name: "Minor bump",
			bump: Minor,
			want: Version{Major: 1, Minor: 3, Patch: 0},
		},
		{
			name: "Major bump",
			bump: Major,
			want: Version{Major: 2, Minor: 0, Patch: 0},
		},
		{
			name: "Unknown bump (no change)",
			bump: "unknown",
			want: base,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := base.Increment(tt.bump)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestString(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3}
	got := v.String()
	want := "1.2.3"

	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestLessThan(t *testing.T) {
	tests := []struct {
		a, b Version
		want bool
	}{
		{Version{1, 0, 0}, Version{1, 0, 1}, true},
		{Version{1, 2, 0}, Version{1, 3, 0}, true},
		{Version{1, 2, 3}, Version{2, 0, 0}, true},
		{Version{2, 0, 0}, Version{1, 2, 3}, false},
		{Version{1, 2, 3}, Version{1, 2, 3}, false},
	}

	for _, tt := range tests {
		got := tt.a.LessThan(tt.b)
		if got != tt.want {
			t.Errorf("LessThan(%v, %v) = %v; want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseVersionType(t *testing.T) {
	tests := []struct {
		input    string
		expected VersionType
		hasError bool
	}{
		{"major", Major, false},
		{"minor", Minor, false},
		{"patch", Patch, false},
		{"MAJOR", Major, false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		got, err := ParseVersionType(tt.input)
		if tt.hasError && err == nil {
			t.Errorf("ParseVersionType(%q) expected error, got none", tt.input)
		}
		if !tt.hasError && got != tt.expected {
			t.Errorf("ParseVersionType(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestVersionTypeString(t *testing.T) {
	tests := []struct {
		v    VersionType
		want string
	}{
		{Major, "Major"},
		{Minor, "Minor"},
		{Patch, "Patch"},
		{VersionType("unknown"), "unknown"},
	}

	for _, tt := range tests {
		got := tt.v.String()
		if got != tt.want {
			t.Errorf("VersionType(%q).String() = %q; want %q", tt.v, got, tt.want)
		}
	}
}
