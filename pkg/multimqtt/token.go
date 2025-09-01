/*
 * Copyright 2025 InfAI (CC SES)
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

package multimqtt

import (
	"context"
	"errors"
	"time"
)

type Token struct {
	ctx context.Context
}

func NewToken() (token *Token, finish func(err error)) {
	ctx, cancel := context.WithCancelCause(context.Background())
	return &Token{ctx: ctx}, cancel
}

func (this *Token) Wait() bool {
	<-this.ctx.Done()
	return true
}

func (this *Token) WaitTimeout(duration time.Duration) bool {
	ctx, _ := context.WithTimeout(this.ctx, duration)
	<-ctx.Done()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return false
	}
	return true
}

func (this *Token) Done() <-chan struct{} {
	return this.ctx.Done()
}

func (this *Token) Error() error {
	err := this.ctx.Err()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
