/*
 * Cellarium Quests — push notification subscription
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
    const btn = document.getElementById('push-toggle');
    const testBtn = document.getElementById('push-test');
    if (!btn || !('serviceWorker' in navigator) || !('PushManager' in window)) return;
    async function getKey() {
        const r = await fetch('/api/push/vapid-public-key');
        const b64 = await r.text();
        const raw = atob(b64.replace(/-/g, '+').replace(/_/g, '/'));
        const arr = new Uint8Array(raw.length);
        for (let i = 0; i < raw.length; i++) arr[i] = raw.charCodeAt(i);
        return arr;
    }
    async function subscribe() {
        const reg = await navigator.serviceWorker.ready;
        const sub = await reg.pushManager.subscribe({userVisibleOnly: true, applicationServerKey: await getKey()});
        const j = sub.toJSON();
        const resp = await fetch('/api/push/subscribe', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({endpoint: j.endpoint, keys: j.keys})});
        if (!resp.ok) {
            await sub.unsubscribe();
            throw new Error('Server failed to save subscription: ' + resp.status);
        }
        btn.textContent = 'Disable Notifications'; btn.dataset.active = '1';
        if (testBtn) testBtn.style.display = '';
    }
    async function unsubscribe() {
        const reg = await navigator.serviceWorker.ready;
        const sub = await reg.pushManager.getSubscription();
        if (sub) {
            const resp = await fetch('/api/push/unsubscribe', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({endpoint: sub.endpoint})});
            if (!resp.ok) {
                throw new Error('Server failed to remove subscription: ' + resp.status);
            }
            await sub.unsubscribe();
        }
        btn.textContent = 'Enable Notifications'; btn.dataset.active = '0';
        if (testBtn) testBtn.style.display = 'none';
    }
    navigator.serviceWorker.register('/service-worker.js').then(async () => {
        const reg = await navigator.serviceWorker.ready;
        const sub = await reg.pushManager.getSubscription();
        if (sub) {
            btn.textContent = 'Disable Notifications'; btn.dataset.active = '1';
            if (testBtn) testBtn.style.display = '';
        }
    });
    btn.addEventListener('click', async () => {
        try {
            if (btn.dataset.active === '1') { await unsubscribe(); } else { await subscribe(); }
        } catch (err) {
            console.error('Push notification error:', err);
            alert('Failed to update notification settings: ' + err.message);
        }
    });
    if (testBtn) {
        testBtn.addEventListener('click', async () => {
            testBtn.disabled = true;
            testBtn.textContent = 'Sending...';
            try {
                const resp = await fetch('/api/push/test', {method: 'POST'});
                if (!resp.ok) {
                    alert('Error: ' + (await resp.text()).trim());
                    return;
                }
                const data = await resp.json();
                const summary = data.map(r => r.endpoint.slice(-30) + ': ' + (r.error || r.status_code)).join('\n');
                alert('Results:\n' + summary);
            } catch (err) {
                alert('Failed: ' + err.message);
            } finally {
                testBtn.disabled = false;
                testBtn.textContent = 'Test Notification';
            }
        });
    }
}());
