/*
 * Cellarium Quests — drag-and-drop quest reordering
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
    let dragEl = null;
    function initList(list) {
        const items = () => [...list.querySelectorAll('[data-quest-id]')];
        items().forEach(item => {
            item.setAttribute('draggable', 'true');
            item.addEventListener('dragstart', e => { dragEl = item; e.dataTransfer.effectAllowed = 'move'; });
            item.addEventListener('dragover', e => { e.preventDefault(); item.classList.add('drag-over'); });
            item.addEventListener('dragleave', () => item.classList.remove('drag-over'));
            item.addEventListener('drop', e => {
                e.preventDefault(); item.classList.remove('drag-over');
                if (dragEl && dragEl !== item) {
                    const all = items(); const fi = all.indexOf(dragEl); const ti = all.indexOf(item);
                    if (fi < ti) { list.insertBefore(dragEl, item.nextSibling); } else { list.insertBefore(dragEl, item); }
                    saveOrder(list);
                }
                dragEl = null;
            });
            item.addEventListener('dragend', () => { dragEl = null; items().forEach(i => i.classList.remove('drag-over')); });
        });
    }
    function saveOrder(list) {
        [...list.querySelectorAll('[data-quest-id]')].forEach((item, idx) => {
            fetch('/quests/reorder', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: parseInt(item.dataset.questId, 10), sort_order: idx}) }).catch(console.error);
        });
    }
    document.querySelectorAll('.quest-list').forEach(initList);
}());
