/*
 * Copyright 2021 InfAI (CC SES)
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

package main

import (
	"context"
	"flag"
	"github.com/SENERGY-Platform/api-docs-provider/lib/client"
	"github.com/SENERGY-Platform/process-sync/docs"
	"github.com/SENERGY-Platform/process-sync/pkg/api"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/controller"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
)

func main() {
	time.Sleep(5 * time.Second) //wait for routing tables in cluster

	confLocation := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	config, err := configuration.Load(*confLocation)
	if err != nil {
		log.Fatal("ERROR: unable to load config ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	ctrl, err := controller.NewDefault(config, ctx)
	if err != nil {
		debug.PrintStack()
		log.Fatal("FATAL:", err)
	}

	err = api.Start(config, ctx, ctrl)
	if err != nil {
		debug.PrintStack()
		log.Fatal("FATAL:", err)
	}

	if config.CleanupInterval != "" && config.CleanupInterval != "-" && config.CleanupMaxAge != "" && config.CleanupMaxAge != "-" {
		go cleanup(ctx, ctrl, config)
	}

	if config.ApiDocsProviderBaseUrl != "" && config.ApiDocsProviderBaseUrl != "-" {
		err = PublishAsyncApiDoc(config)
		if err != nil {
			log.Fatal(err)
		}
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	sig := <-shutdown
	log.Println("received shutdown signal", sig)
	cancel()
	time.Sleep(1 * time.Second) //give connections time to close gracefully
}

func cleanup(ctx context.Context, ctrl *controller.Controller, config configuration.Config) {
	cleanupInterval, err := time.ParseDuration(config.CleanupInterval)
	if err != nil {
		log.Println("WARNING: invalid CleanupInterval", config.CleanupInterval, err)
		return
	}
	maxAge, err := time.ParseDuration(config.CleanupMaxAge)
	if err != nil {
		log.Println("WARNING: invalid CleanupMaxAge", config.CleanupMaxAge, err)
		return
	}

	err = ctrl.RemoveOldEntities(maxAge)
	if err != nil {
		log.Println("WARNING: RemoveOldEntities() ->", err)
	}
	t := time.NewTicker(cleanupInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			err = ctrl.RemoveOldEntities(maxAge)
			if err != nil {
				log.Println("WARNING: RemoveOldEntities() ->", err)
			}
		}
	}
}

func PublishAsyncApiDoc(conf configuration.Config) error {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	return client.New(http.DefaultClient, conf.ApiDocsProviderBaseUrl).AsyncapiPutDoc(ctx, "github_com_SENERGY-Platform_process-sync", docs.AsyncApiDoc)
}
