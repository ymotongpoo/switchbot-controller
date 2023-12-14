// Copyright 2023 Yoshi Yamaguchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	openToken string

	secretKey string
)

func init() {
	if err := godotenv.Load(); err != nil {
		slog.Error(fmt.Sprintf("failed to load .env file: %v", err))
		os.Exit(1)
	}
	openToken = os.Getenv("SWITCHBOT_TOKEN")
	secretKey = os.Getenv("SWITCHBOT_SECRET")

	if openToken == "" {
		slog.Error("add SWITCHBOT_TOKEN environment variable in .env")
		os.Exit(1)
	}
	if secretKey == "" {
		slog.Error("add SWITCHBOT_SECRET environment variable in .env")
		os.Exit(1)
	}
}

func main() {
	c := NewSwitchBotController(openToken, secretKey)
	c.refreshDevices()

	http.HandleFunc("/devices", c.deviceListHandler)
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":18888", nil); err != nil {
		slog.Error(fmt.Sprintf("error running HTTP server: %v", err))
		os.Exit(1)
		return
	}
}
