// Cellarium Pockets — colour validation tests
// Copyright (C) 2026 Maroš Kučera
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import "testing"

func TestValidColour(t *testing.T) {
	valid := []string{"ruby", "amber", "olive", "forest", "teal", "steel", "plum", "rose"}
	for _, c := range valid {
		if !validColour(c) {
			t.Errorf("validColour(%q) = false, want true", c)
		}
	}

	invalid := []string{"", "red", "blue", "Ruby", "RUBY", "#b34d4d", "unknown"}
	for _, c := range invalid {
		if validColour(c) {
			t.Errorf("validColour(%q) = true, want false", c)
		}
	}
}

func TestColourHex(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"ruby", "#b34d4d"},
		{"amber", "#a07030"},
		{"olive", "#7a8a30"},
		{"forest", "#4a8a50"},
		{"teal", "#3a8080"},
		{"steel", "#4a6a9a"},
		{"plum", "#7a5a90"},
		{"rose", "#9a5070"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := colourHex(tt.key)
			if got != tt.want {
				t.Errorf("colourHex(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestColourHexUnknown(t *testing.T) {
	got := colourHex("unknown")
	if got != "#000000" {
		t.Errorf("colourHex(unknown) = %q, want #000000", got)
	}
}

func TestAllColours(t *testing.T) {
	colours := allColours()
	if len(colours) != 8 {
		t.Fatalf("expected 8 colours, got %d", len(colours))
	}
	// Verify order matches spec
	expected := []string{"ruby", "amber", "olive", "forest", "teal", "steel", "plum", "rose"}
	for i, c := range colours {
		if c.Key != expected[i] {
			t.Errorf("colour[%d].Key = %q, want %q", i, c.Key, expected[i])
		}
		if c.Hex == "" {
			t.Errorf("colour[%d].Hex is empty", i)
		}
		if c.Name == "" {
			t.Errorf("colour[%d].Name is empty", i)
		}
	}
}
