// Cellarium Quests — background ticker for reminders and failure detection
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
	"context"
	"encoding/json"
	"log"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

type notifyConfig struct {
	VAPIDPrivateKey string
	VAPIDPublicKey  string
	VAPIDSubject    string
}

// startTicker runs the background goroutine that checks reminders and failures.
// Call with go startTicker(ctx, q, cfg).
func startTicker(ctx context.Context, q sqlc.Querier, tx txRunner, cfg notifyConfig) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			runTickerTasks(ctx, q, tx, cfg, t)
		}
	}
}

func runTickerTasks(ctx context.Context, q sqlc.Querier, tx txRunner, cfg notifyConfig, now time.Time) {
	today := localToday(now)
	if err := ensureFailedQuests(ctx, tx, today); err != nil {
		log.Printf("ensureFailedQuests error: %v", err)
	}
	if cfg.VAPIDPrivateKey == "" {
		return
	}
	nowTime := localTime(now)
	dueQuests, err := q.ListDueReminders(ctx, sqlc.ListDueRemindersParams{
		NowTime: nowTime,
		Today:   today,
	})
	if err != nil {
		log.Printf("ListDueReminders error: %v", err)
		return
	}
	if len(dueQuests) == 0 {
		return
	}
	subs, err := q.ListPushSubscriptions(ctx)
	if err != nil {
		log.Printf("ListPushSubscriptions error: %v", err)
		return
	}
	for _, quest := range dueQuests {
		payload, _ := json.Marshal(map[string]string{
			"title": quest.Title,
			"body":  "Quest reminder",
		})
		for _, sub := range subs {
			sendPush(sub, string(payload), cfg)
		}
		if err := q.MarkReminderSent(ctx, sqlc.MarkReminderSentParams{
			Today: today,
			ID:    quest.ID,
		}); err != nil {
			log.Printf("MarkReminderSent error: %v", err)
		}
	}
}

func sendPush(sub sqlc.QuestsPushSubscription, payload string, cfg notifyConfig) {
	_, err := webpush.SendNotification([]byte(payload), &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			Auth:   sub.Auth,
			P256dh: sub.P256dh,
		},
	}, &webpush.Options{
		VAPIDPrivateKey: cfg.VAPIDPrivateKey,
		VAPIDPublicKey:  cfg.VAPIDPublicKey,
		Subscriber:      cfg.VAPIDSubject,
		TTL:             30,
	})
	if err != nil {
		log.Printf("sendPush error: %v", err)
	}
}
