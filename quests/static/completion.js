// Cellarium Quests — checkbox-based quest completion
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

(function () {
    document.querySelectorAll('.quest-complete-cb').forEach(function (cb) {
        cb.addEventListener('change', function () {
            var id = cb.dataset.questId;
            var completing = cb.checked;
            var action = completing ? 'complete' : 'uncomplete';
            var card = cb.closest('.quest-card');

            fetch('/quests/' + id + '/' + action, { method: 'POST' })
                .then(function (res) {
                    if (res.ok) {
                        if (card) {
                            if (completing) {
                                card.classList.add('status-completed');
                            } else {
                                card.classList.remove('status-completed');
                            }
                        }
                    } else {
                        cb.checked = !completing;
                    }
                })
                .catch(function () {
                    cb.checked = !completing;
                });
        });
    });
}());
