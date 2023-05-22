/*
 * Copyright (c) 2023 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package canary

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

func (this *Canary) testNotification(wg *sync.WaitGroup, token string) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		text := "canary-notification-" + time.Now().String()

		err := this.sendNotification(token, text)
		if err != nil {
			return
		}

		time.Sleep(this.getChangeGuaranteeDuration())

		notifications, err := this.getNotifications(token)
		if err != nil {
			return
		}

		ids := []string{}
		found := false
		for _, n := range notifications {
			ids = append(ids, n.Id)
			if n.Message == text {
				found = true
			}
		}

		if !found {
			this.metrics.UnexpectedNotificationStateErr.Inc()
			log.Printf("UnexpectedNotificationStateErr: %#v\n", found)
		}

		err = this.deleteNotifications(token, ids)
		if err != nil {
			return
		}
	}()
}

type Message struct {
	UserId  string `json:"userId" bson:"userId"`
	Title   string `json:"title" bson:"title"`
	Message string `json:"message" bson:"message"`
}

type Notification struct {
	Id        string    `json:"_id" bson:"_id"`
	UserId    string    `json:"userId" bson:"userId"`
	Title     string    `json:"title" bson:"title"`
	Message   string    `json:"message" bson:"message"`
	IsRead    bool      `json:"isRead" bson:"isRead"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type NotificationList struct {
	Total         int64          `json:"total"`
	Limit         int            `json:"limit"`
	Offset        int            `json:"offset"`
	Notifications []Notification `json:"notifications"`
}

func (this *Canary) sendNotification(token string, text string) (err error) {
	message := Message{
		Title:   "Canary-Test-Message",
		Message: text,
	}

	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(message)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", this.config.NotificationUrl+"/notifications", b)
	if err != nil {
		log.Println("ERROR: unable to send notification", err)
		return err
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	this.metrics.NotificationPublishCount.Inc()
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	this.metrics.NotificationPublishLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		log.Println("ERROR: unable to send notification", err)
		this.metrics.NotificationPublishErr.Inc()
		return err
	}
	defer resp.Body.Close()
	respMsg, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		this.metrics.NotificationPublishErr.Inc()
		log.Println("ERROR: unexpected response status from notifier", resp.StatusCode, string(respMsg))
		return errors.New("unexpected response status from notifier " + resp.Status)
	}
	return nil
}

func (this *Canary) getNotifications(token string) (result []Notification, err error) {
	req, err := http.NewRequest("GET", this.config.NotificationUrl+"/notifications", nil)
	if err != nil {
		log.Println("ERROR: unable to send notification", err)
		return result, err
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	req.WithContext(ctx)
	req.Header.Set("Authorization", token)

	this.metrics.NotificationReadCount.Inc()
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	this.metrics.NotificationReadLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		log.Println("ERROR: unable to read notification", err)
		this.metrics.NotificationReadErr.Inc()
		return result, err
	}
	defer resp.Body.Close()
	respMsg, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		this.metrics.NotificationReadErr.Inc()
		log.Println("ERROR: unexpected response status from notifier", resp.StatusCode, string(respMsg))
		return result, errors.New("unexpected response status from notifier " + resp.Status)
	}
	temp := NotificationList{}
	err = json.NewDecoder(resp.Body).Decode(&temp)
	if err != nil {
		log.Println("ERROR: unable to read notification", err)
		this.metrics.NotificationReadErr.Inc()
		return result, err
	}

	return temp.Notifications, nil
}

func (this *Canary) deleteNotifications(token string, ids []string) (err error) {
	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(ids)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", this.config.NotificationUrl+"/notifications", b)
	if err != nil {
		log.Println("ERROR: unable to send notification", err)
		return err
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	this.metrics.NotificationDeleteCount.Inc()
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	this.metrics.NotificationDeleteLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		log.Println("ERROR: unable to send notification", err)
		this.metrics.NotificationDeleteErr.Inc()
		return err
	}
	defer resp.Body.Close()
	respMsg, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		this.metrics.NotificationDeleteErr.Inc()
		log.Println("ERROR: unexpected response status from notifier", resp.StatusCode, string(respMsg))
		return errors.New("unexpected response status from notifier " + resp.Status)
	}
	return nil
}
