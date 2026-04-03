// Cellarium Quests — local timezone helpers
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
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

var appLocation *time.Location

func initLocation() {
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "Europe/Bratislava"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Fatalf("failed to load timezone %q: %v", tz, err)
	}
	appLocation = loc
}

func localToday(now time.Time) pgtype.Date {
	local := now.In(appLocation)
	y, m, d := local.Date()
	return pgtype.Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC), Valid: true}
}

// localTime returns the local time truncated to minute precision,
// matching the reminder_time column semantics and the 60s ticker interval.
func localTime(now time.Time) pgtype.Time {
	local := now.In(appLocation)
	h, min, _ := local.Clock()
	return pgtype.Time{
		Microseconds: int64(h)*3600_000_000 + int64(min)*60_000_000,
		Valid:        true,
	}
}
