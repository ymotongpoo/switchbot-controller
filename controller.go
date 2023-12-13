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
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	switchbot "github.com/nasa9084/go-switchbot/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	apimetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

const meterName = "github.com/ymotongpoo/switchbot-controller"

// SwitchBotController holds a client for SwitchBot and
type SwitchBotController struct {
	cli   *switchbot.Client
	ctx   context.Context
	pdevs []switchbot.Device
	idevs []switchbot.InfraredDevice

	tempGuage  apimetric.Float64ObservableGauge
	humidGuage apimetric.Int64ObservableGauge
}

// NewSwitchBotController initialize a controller instance with given token and secret.
// Also it holds OpenTelemetry metrics related objects.
func NewSwitchBotController(openToken, secretkey string, opts ...switchbot.Option) *SwitchBotController {
	sbc := &SwitchBotController{
		cli: switchbot.New(openToken, secretkey, opts...),
		ctx: context.Background(),
	}

	exporter, err := prometheus.New()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to initialize Prometheus exporter: %v", err))
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	meter := provider.Meter(meterName)

	sbc.tempGuage, err = meter.Float64ObservableGauge(
		"temperature",
		apimetric.WithDescription("temperarature"),
		apimetric.WithUnit("C"),
		apimetric.WithFloat64Callback(sbc.tempObserver),
	)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to initialize temperature guage: %v", err))
	}
	sbc.humidGuage, err = meter.Int64ObservableGauge(
		"humidity",
		apimetric.WithDescription("humidity"),
		apimetric.WithUnit("%"),
		apimetric.WithInt64Callback(sbc.humidObserver),
	)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to initialize humidity guage: %v", err))
	}

	return sbc
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
			buf.WriteString(fmt.Sprintf("id: %v, temperature: %v, humidity: %v\n", d.ID, s.Temperature, s.Humidity))
		default:
		}
	}
	buf.WriteTo(w)
}

func (c *SwitchBotController) tempObserver(_ context.Context, o apimetric.Float64Observer) error {
	svc := c.cli.Device()
	var errs error
	for _, d := range c.pdevs {
		switch d.Type {
		case switchbot.Hub2, switchbot.WoIOSensor:
			s, err := svc.Status(c.ctx, d.ID)
			if err != nil {
				slog.Info(fmt.Sprintf("failed to fetch status of %v", d.ID))
				errs = errors.Join(errs, err)
				continue
			}
			o.Observe(
				s.Temperature,
				apimetric.WithAttributes(
					attribute.String("id", d.ID),
					attribute.String("name", d.Name),
				),
			)
		default:
		}
	}
	return errs
}

func (c *SwitchBotController) humidObserver(_ context.Context, o apimetric.Int64Observer) error {
	svc := c.cli.Device()
	var errs error
	for _, d := range c.pdevs {
		switch d.Type {
		case switchbot.Hub2, switchbot.WoIOSensor:
			s, err := svc.Status(c.ctx, d.ID)
			if err != nil {
				slog.Info(fmt.Sprintf("failed to fetch status of %v", d.ID))
				errs = errors.Join(errs, err)
				continue
			}
			o.Observe(
				int64(s.Humidity),
				apimetric.WithAttributes(
					attribute.String("id", d.ID),
					attribute.String("name", d.Name),
				),
			)
		default:
		}
	}
	return errs
}
