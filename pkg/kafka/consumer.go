/*
 * Copyright 2020 InfAI (CC SES)
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

package kafka

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/segmentio/kafka-go"
	"io"
	"log"
	"time"
)

var FetchError = errors.New("unable to fetch from kafka")
var HandlerError = errors.New("unable to handle kafka message")
var CommitError = errors.New("unable to commit to kafka")

func NewConsumer(ctx context.Context, config configuration.Config, topic string, listener func(delivery []byte) error, errorhandler func(err error) (fatal bool)) error {
	broker, err := GetBroker(config.KafkaUrl)
	if err != nil {
		log.Println("ERROR: unable to get broker list", err)
		return err
	}

	err = InitTopic(config.KafkaUrl, topic)
	if err != nil {
		log.Println("ERROR: unable to create topic", err)
		return err
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		CommitInterval: 0, //synchronous commits
		Brokers:        broker,
		GroupID:        config.KafkaConsumerGroup,
		Topic:          topic,
		MaxWait:        1 * time.Second,
		Logger:         log.New(io.Discard, "", 0),
		ErrorLogger:    log.New(io.Discard, "", 0),
	})
	go func() {
		defer r.Close()
		defer log.Println("close consumer for topic ", topic)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				m, err := r.FetchMessage(ctx)
				if err == io.EOF || err == context.Canceled {
					return
				}
				if err != nil {
					fatal := errorhandler(fmt.Errorf("%w: %v", FetchError, err.Error()))
					if fatal {
						return
					}
				}

				err = retry(func() error {
					return listener(m.Value)
				}, func(n int64) time.Duration {
					return time.Duration(n) * time.Second
				}, 10*time.Minute)

				if err != nil {
					fatal := errorhandler(fmt.Errorf("%w: %v", HandlerError, err.Error()))
					if fatal {
						return
					}
				} else {
					err = r.CommitMessages(ctx, m)
					if err != nil {
						fatal := errorhandler(fmt.Errorf("%w: %v", CommitError, err.Error()))
						if fatal {
							return
						}
					}
				}

			}
		}
	}()
	return nil
}

func retry(f func() error, waitProvider func(n int64) time.Duration, timeout time.Duration) (err error) {
	err = errors.New("")
	start := time.Now()
	for i := int64(1); err != nil && time.Since(start) < timeout; i++ {
		err = f()
		if err != nil {
			log.Println("ERROR: kafka listener error:", err)
			wait := waitProvider(i)
			if time.Since(start)+wait < timeout {
				log.Println("ERROR: retry after:", wait.String())
				time.Sleep(wait)
			} else {
				return err
			}
		}
	}
	return err
}
