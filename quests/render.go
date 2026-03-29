// Cellarium Quests — template rendering helper
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

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
)

// renderTemplate executes the named template into a buffer. If rendering fails,
// it logs the error and sends a 500 response. Returns true if rendering succeeded.
func renderTemplate(w http.ResponseWriter, tmpl *template.Template, name string, data any) bool {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("template %q error: %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return false
	}
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("response write error: %v", err)
	}
	return true
}
