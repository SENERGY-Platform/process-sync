/*
 * Copyright 2019 InfAI (CC SES)
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
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/SENERGY-Platform/process-deployment/lib/interfaces"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	ctx    context.Context
}

type DebugLogPrinter struct {
	log *slog.Logger
}

func (this *DebugLogPrinter) Printf(s string, i ...interface{}) {
	this.log.Debug(fmt.Sprintf(s, i...))
}

func NewProducer(ctx context.Context, kafkaUrl string, topic string, logger *slog.Logger, topicInit bool) (interfaces.Producer, error) {
	result := &Producer{ctx: ctx}
	broker, err := GetBroker(kafkaUrl)
	if err != nil {
		logger.Error("unable to get broker list", "error", err)
		return nil, err
	}
	if topicInit {
		err = InitTopic(kafkaUrl, topic)
		if err != nil {
			logger.Error("unable to create topic", "error", err)
			return nil, err
		}
	}
	result.writer = &kafka.Writer{
		Addr:        kafka.TCP(broker...),
		Topic:       topic,
		Logger:      &DebugLogPrinter{log: logger},
		Async:       false,
		BatchSize:   1,
		Balancer:    &kafka.Hash{},
		ErrorLogger: log.New(os.Stderr, "KAFKA", 0),
	}

	go func() {
		<-ctx.Done()
		result.writer.Close()
	}()
	return result, nil
}

func (this *Producer) Produce(key string, message []byte) error {
	return this.writer.WriteMessages(this.ctx, kafka.Message{
		Key:   []byte(key),
		Value: message,
		Time:  time.Now(),
	})
}
