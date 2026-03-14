/*
 * Cellarium Receipt Tracker — client-side application logic
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

(function () {
  "use strict";

  const DB_NAME = "receipt-tracker";
  const STORE_NAME = "offline-queue";
  const DB_VERSION = 1;

  const form = document.getElementById("entry-form");
  const dateInput = document.getElementById("entry_date");
  const statusText = document.getElementById("status-text");
  const queueCount = document.getElementById("queue-count");
  const toastEl = document.getElementById("toast");

  // Default date to today
  dateInput.value = new Date().toISOString().slice(0, 10);

  // IndexedDB helpers
  function openDB() {
    return new Promise((resolve, reject) => {
      const req = indexedDB.open(DB_NAME, DB_VERSION);
      req.onupgradeneeded = () => {
        req.result.createObjectStore(STORE_NAME, {
          keyPath: "id",
          autoIncrement: true,
        });
      };
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });
  }

  async function enqueue(entry) {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_NAME, "readwrite");
      tx.objectStore(STORE_NAME).add(entry);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
    });
  }

  async function dequeueAll() {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_NAME, "readwrite");
      const store = tx.objectStore(STORE_NAME);
      const req = store.getAll();
      req.onsuccess = () => {
        store.clear();
        resolve(req.result);
      };
      req.onerror = () => reject(req.error);
    });
  }

  async function getQueueSize() {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STORE_NAME, "readonly");
      const req = tx.objectStore(STORE_NAME).count();
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });
  }

  // Toast
  let toastTimer;
  function showToast(message, type) {
    clearTimeout(toastTimer);
    toastEl.textContent = message;
    toastEl.className = "toast " + type;
    toastTimer = setTimeout(() => {
      toastEl.className = "toast hidden";
    }, 3000);
  }

  // Status
  async function updateStatus() {
    const online = navigator.onLine;
    statusText.textContent = online ? "Online" : "Offline";
    const count = await getQueueSize();
    if (count > 0) {
      queueCount.textContent = count + " queued";
      queueCount.classList.remove("hidden");
    } else {
      queueCount.classList.add("hidden");
    }
  }

  // Submit entry to API
  async function submitEntry(entry) {
    const resp = await fetch("/api/entries", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(entry),
    });
    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(text.trim() || "Request failed");
    }
    return resp.json();
  }

  // Sync offline queue
  async function syncQueue() {
    const entries = await dequeueAll();
    const failed = [];
    for (const entry of entries) {
      try {
        await submitEntry(entry);
      } catch {
        failed.push(entry);
      }
    }
    // Re-enqueue failures
    for (const entry of failed) {
      await enqueue(entry);
    }
    await updateStatus();
    if (entries.length > 0 && failed.length === 0) {
      showToast("Synced " + entries.length + " queued entries", "success");
    }
  }

  // Form submit
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const valueInput = document.getElementById("value");
    const noteInput = document.getElementById("note");

    const entry = {
      value: valueInput.value.trim().replace(",", "."),
      entry_date: dateInput.value || undefined,
      note: noteInput.value.trim() || undefined,
    };

    if (!entry.value) return;

    const btn = form.querySelector("button[type=submit]");
    btn.disabled = true;

    try {
      if (navigator.onLine) {
        await submitEntry(entry);
        showToast("Entry saved", "success");
      } else {
        await enqueue(entry);
        showToast("Queued for sync", "success");
      }
      valueInput.value = "";
      noteInput.value = "";
      dateInput.value = new Date().toISOString().slice(0, 10);
      valueInput.focus();
    } catch (err) {
      // If online submit fails, queue it
      await enqueue(entry);
      showToast("Queued (server error)", "error");
    } finally {
      btn.disabled = false;
      await updateStatus();
    }
  });

  // Online/offline listeners
  window.addEventListener("online", () => {
    updateStatus();
    syncQueue();
  });
  window.addEventListener("offline", updateStatus);

  // Service worker registration
  if ("serviceWorker" in navigator) {
    navigator.serviceWorker.register("sw.js");
  }

  // Init
  updateStatus();
})();
