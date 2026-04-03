// Cellarium Quests — timezone function tests
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
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestLocalToday_UTC(t *testing.T) {
	origLoc := appLocation
	defer func() { appLocation = origLoc }()
	appLocation = time.UTC
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	got := localToday(now)

	want := pgtype.Date{Time: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), Valid: true}
	if got != want {
		t.Errorf("localToday(%v) = %v, want %v", now, got, want)
	}
}

func TestLocalToday_CETBeforeMidnight(t *testing.T) {
	origLoc := appLocation
	defer func() { appLocation = origLoc }()
	loc, err := time.LoadLocation("Europe/Bratislava")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}
	appLocation = loc

	// 2026-01-15 23:30 UTC → CET (UTC+1 in winter) → 2026-01-16 00:30
	now := time.Date(2026, 1, 15, 23, 30, 0, 0, time.UTC)

	got := localToday(now)

	want := pgtype.Date{Time: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC), Valid: true}
	if got != want {
		t.Errorf("localToday(%v) = %v, want %v", now, got, want)
	}
}

func TestLocalToday_CESTBeforeMidnight(t *testing.T) {
	origLoc := appLocation
	defer func() { appLocation = origLoc }()
	loc, err := time.LoadLocation("Europe/Bratislava")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}
	appLocation = loc

	// 2026-07-15 22:30 UTC → CEST (UTC+2 in summer) → 2026-07-16 00:30
	now := time.Date(2026, 7, 15, 22, 30, 0, 0, time.UTC)

	got := localToday(now)

	want := pgtype.Date{Time: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC), Valid: true}
	if got != want {
		t.Errorf("localToday(%v) = %v, want %v", now, got, want)
	}
}

func TestLocalTime_convertsToLocal(t *testing.T) {
	origLoc := appLocation
	defer func() { appLocation = origLoc }()
	loc, err := time.LoadLocation("Europe/Bratislava")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}
	appLocation = loc

	// 2026-01-15 22:30 UTC → CET (UTC+1) → 23:30 local
	now := time.Date(2026, 1, 15, 22, 30, 0, 0, time.UTC)

	got := localTime(now)

	wantMicroseconds := int64(23)*3600_000_000 + int64(30)*60_000_000
	want := pgtype.Time{Microseconds: wantMicroseconds, Valid: true}
	if got != want {
		t.Errorf("localTime(%v) = %v, want %v", now, got, want)
	}
}

func TestInitLocation_default(t *testing.T) {
	origLoc := appLocation
	defer func() { appLocation = origLoc }()
	oldTZ, hadTZ := os.LookupEnv("TZ")
	os.Unsetenv("TZ")
	defer func() {
		if hadTZ {
			os.Setenv("TZ", oldTZ)
		} else {
			os.Unsetenv("TZ")
		}
	}()

	initLocation()

	if appLocation == nil {
		t.Fatal("appLocation is nil after initLocation()")
	}
	if appLocation.String() != "Europe/Bratislava" {
		t.Errorf("appLocation = %q, want %q", appLocation.String(), "Europe/Bratislava")
	}
}

func TestInitLocation_envOverride(t *testing.T) {
	origLoc := appLocation
	defer func() { appLocation = origLoc }()
	old, hadOld := os.LookupEnv("TZ")
	os.Setenv("TZ", "America/New_York")
	defer func() {
		if hadOld {
			os.Setenv("TZ", old)
		} else {
			os.Unsetenv("TZ")
		}
	}()

	initLocation()

	if appLocation == nil {
		t.Fatal("appLocation is nil after initLocation()")
	}
	if appLocation.String() != "America/New_York" {
		t.Errorf("appLocation = %q, want %q", appLocation.String(), "America/New_York")
	}
}
