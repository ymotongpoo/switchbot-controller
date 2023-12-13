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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	switchbot "github.com/nasa9084/go-switchbot/v3"
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

type SwitchBotController struct {
	cli   *switchbot.Client
	ctx   context.Context
	pdevs []switchbot.Device
	idevs []switchbot.InfraredDevice
}

func NewSwitchBotController(openToken, secretkey string, opts ...switchbot.Option) *SwitchBotController {
	return &SwitchBotController{
		cli: switchbot.New(openToken, secretkey, opts...),
		ctx: context.Background(),
	}
}

func (c *SwitchBotController) refreshDevices() error {
	svc := c.cli.Device()
	pdevs, idevs, err := svc.List(c.ctx)
	if err != nil {
		return err
	}
	c.pdevs = pdevs
	c.idevs = idevs
	return nil
}

func (c *SwitchBotController) deviceListHandler(w http.ResponseWriter, r *http.Request) {
	c.refreshDevices()

	var buf bytes.Buffer
	for _, d := range c.pdevs {
		b, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			continue
		}
		buf.Write(b)
	}
	buf.WriteTo(w)
}

func (c *SwitchBotController) metricsHandler(w http.ResponseWriter, r *http.Request) {
	svc := c.cli.Device()

	var buf bytes.Buffer
	for _, d := range c.pdevs {
		switch d.Type {
		case switchbot.Hub2, switchbot.WoIOSensor:
			s, err := svc.Status(c.ctx, d.ID)
			if err != nil {
				slog.Info(fmt.Sprintf("failed to fetch status of %v", d.ID))
				continue
			}
			buf.WriteString(fmt.Sprintf("id: %v, temperature: %v, humidity: %v", d.ID, s.Temperature, s.Humidity))
		default:
		}
	}
	buf.WriteTo(w)
}

func main() {
	c := NewSwitchBotController(openToken, secretKey)
	c.refreshDevices()

	http.HandleFunc("/devices", c.deviceListHandler)
	http.HandleFunc("/metrics", c.metricsHandler)
	if err := http.ListenAndServe(":8888", nil); err != nil {
		slog.Error(fmt.Sprintf("error running HTTP server: %v", err))
		os.Exit(1)
		return
	}
}
