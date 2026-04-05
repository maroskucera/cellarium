/*
 * Cellarium Quests — quest giver autocomplete
 * Copyright (C) 2026 Maroš Kučera
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */
(function() {
    'use strict';
    const dl = document.getElementById('quest-givers-list');
    if (!dl) return;
    fetch('/api/quest-givers').then(r => r.json()).then(givers => {
        givers.forEach(g => { const opt = document.createElement('option'); opt.value = g; dl.appendChild(opt); });
    }).catch(console.error);
}());
