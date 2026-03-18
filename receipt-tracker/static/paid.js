/*
 * Cellarium Receipt Tracker — checkbox toggle logic for mark-as-paid page
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

document.addEventListener("DOMContentLoaded", function () {
  var selectAll = document.getElementById("select-all");
  var batchToggles = document.querySelectorAll(".batch-toggle");
  var entryCbs = document.querySelectorAll(".entry-cb");

  function updateBatchToggle(batch) {
    var entries = document.querySelectorAll('.entry-cb[data-batch="' + batch + '"]');
    var toggle = document.querySelector('.batch-toggle[data-batch="' + batch + '"]');
    if (!toggle) return;
    toggle.checked = Array.prototype.every.call(entries, function (cb) {
      return cb.checked;
    });
  }

  function updateSelectAll() {
    if (!selectAll) return;
    selectAll.checked = entryCbs.length > 0 && Array.prototype.every.call(entryCbs, function (cb) {
      return cb.checked;
    });
  }

  if (selectAll) {
    selectAll.addEventListener("change", function () {
      var checked = selectAll.checked;
      batchToggles.forEach(function (bt) { bt.checked = checked; });
      entryCbs.forEach(function (cb) { cb.checked = checked; });
    });
  }

  batchToggles.forEach(function (batchCb) {
    batchCb.addEventListener("change", function () {
      var batch = batchCb.dataset.batch;
      var entries = document.querySelectorAll('.entry-cb[data-batch="' + batch + '"]');
      entries.forEach(function (cb) { cb.checked = batchCb.checked; });
      updateSelectAll();
    });
  });

  entryCbs.forEach(function (entryCb) {
    entryCb.addEventListener("change", function () {
      updateBatchToggle(entryCb.dataset.batch);
      updateSelectAll();
    });
  });
});
