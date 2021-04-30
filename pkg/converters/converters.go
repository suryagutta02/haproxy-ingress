/*
Copyright 2021 The HAProxy Ingress Controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package converters

import (
	"github.com/jcmoraisjr/haproxy-ingress/pkg/converters/configmap"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/converters/gateway"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/converters/ingress"
	convtypes "github.com/jcmoraisjr/haproxy-ingress/pkg/converters/types"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/haproxy"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/utils"
)

type Config interface {
	Sync()
}

func NewConverter(timer *utils.Timer, haproxy haproxy.Config, options *convtypes.ConverterOptions) Config {
	return &converters{
		timer:   timer,
		haproxy: haproxy,
		options: options,
	}
}

type converters struct {
	timer   *utils.Timer
	haproxy haproxy.Config
	options *convtypes.ConverterOptions
}

func (c *converters) Sync() {
	changed := c.options.Cache.SwapChangedObjects()
	gatewayConverter := gateway.NewGatewayConverter(c.options, c.haproxy, changed)
	ingressConverter := ingress.NewIngressConverter(c.options, c.haproxy, changed)

	needFullSync := changed.NeedFullSync ||
		gatewayConverter.NeedFullSync() ||
		ingressConverter.NeedFullSync()
	if needFullSync {
		c.haproxy.Clear()
	}
	l := len(changed.Objects)
	if l > 100 {
		c.options.Logger.InfoV(2, "applying %d change notifications", l)
	} else if l > 1 {
		c.options.Logger.InfoV(2, "applying %d change notifications: %v", l, changed.Objects)
	} else if l == 1 {
		c.options.Logger.InfoV(2, "applying 1 change notification: %v", changed.Objects)
	}

	//
	// gateway converter
	//
	if c.options.HasGateway {
		gatewayConverter.Sync(needFullSync)
		c.timer.Tick("parse_gateway")
	}

	//
	// ingress converter
	//
	ingressConverter.Sync(needFullSync)
	c.timer.Tick("parse_ingress")

	//
	// configmap converters
	//
	if needFullSync || changed.TCPConfigMapDataNew != nil {
		tcpSvcConverter := configmap.NewTCPServicesConverter(c.options, c.haproxy, changed)
		tcpSvcConverter.Sync()
		c.timer.Tick("parse_tcp_svc")
	}

}
