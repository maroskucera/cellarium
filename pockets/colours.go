// Cellarium Pockets — preset colour constants and validation
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

type colour struct {
	Key  string
	Name string
	Hex  string
}

var colours = []colour{
	{"ruby", "Ruby", "#b34d4d"},
	{"amber", "Amber", "#a07030"},
	{"olive", "Olive", "#7a8a30"},
	{"forest", "Forest", "#4a8a50"},
	{"teal", "Teal", "#3a8080"},
	{"steel", "Steel", "#4a6a9a"},
	{"plum", "Plum", "#7a5a90"},
	{"rose", "Rose", "#9a5070"},
}

var colourMap = func() map[string]colour {
	m := make(map[string]colour, len(colours))
	for _, c := range colours {
		m[c.Key] = c
	}
	return m
}()

func validColour(key string) bool {
	_, ok := colourMap[key]
	return ok
}

func colourHex(key string) string {
	if c, ok := colourMap[key]; ok {
		return c.Hex
	}
	return "#000000"
}

func allColours() []colour {
	cp := make([]colour, len(colours))
	copy(cp, colours)
	return cp
}
