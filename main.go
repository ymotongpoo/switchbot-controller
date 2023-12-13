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
	"context"
	"encoding/json"
	"fmt"
	"log"
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
		log.Fatalf("failed to load .env file: %v", err)
	}
	openToken = os.Getenv("SWITCHBOT_TOKEN")
	secretKey = os.Getenv("SWITCHBOT_SECRET")

	if openToken == "" {
		log.Fatalf("add SWITCHBOT_TOKEN environment variable in .env")
	}
	if secretKey == "" {
		log.Fatalf("add SWITCHBOT_SECRET environment variable in .env")
	}
}

func main() {
	c := switchbot.New(openToken, secretKey)
	ctx := context.Background()
	svc := c.Device()
	ds, _, _ := svc.List(ctx)
	for _, d := range ds {
		b, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			continue
		}
		fmt.Println(string(b))

		switch d.Type {
		case switchbot.Hub2, switchbot.WoIOSensor:
			s, err := svc.Status(ctx, d.ID)
			if err != nil {
				log.Printf("failed to fetch status of %v", d.ID)
				continue
			}
			fmt.Printf("id: %v, temperature: %v, humidity: %v", d.ID, s.Temperature, s.Humidity)
		default:
		}
	}
}
