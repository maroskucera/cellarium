/*
 * Cellarium Quests — service worker for push notifications
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
self.addEventListener('push', event => {
    const data = event.data ? event.data.json() : {title: 'Quest reminder', body: ''};
    event.waitUntil(self.registration.showNotification(data.title, {body: data.body, icon: '/static/icon-192.png'}));
});
self.addEventListener('notificationclick', event => {
    event.notification.close();
    event.waitUntil(clients.openWindow('/'));
});
