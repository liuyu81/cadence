// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package config

import (
	"errors"
	"fmt"
	"github.com/uber/ringpop-go"
	"github.com/uber/ringpop-go/discovery"
	"github.com/uber/ringpop-go/discovery/jsonfile"
	"github.com/uber/ringpop-go/discovery/statichosts"
	"github.com/uber/ringpop-go/swim"
	"github.com/uber/tchannel-go"
	"strings"
	"time"
)

const (
	// BootstrapModeNone represents a bootstrap mode set to nothing or invalid
	BootstrapModeNone BootstrapMode = iota
	// BootstrapModeFile represents a file-based bootstrap mode
	BootstrapModeFile
	// BootstrapModeHosts represents a list of hosts passed in the configuration
	BootstrapModeHosts
)

const (
	defaultMaxJoinDuration = 10 * time.Second
)

// RingpopFactory implements the RingpopFactory interface
type RingpopFactory struct {
	config *Ringpop
}

// NewFactory builds a ringpop factory conforming
// to the underlying configuration
func (rpConfig *Ringpop) NewFactory() (*RingpopFactory, error) {
	return newRingpopFactory(rpConfig)
}

func (rpConfig *Ringpop) validate() error {
	if len(rpConfig.Name) == 0 {
		return fmt.Errorf("ringpop config missing `name` param")
	}
	if err := validateBootstrapMode(rpConfig); err != nil {
		return err
	}
	return nil
}

// UnmarshalYAML is called by the yaml package to convert
// the config YAML into a BootstrapMode.
func (m *BootstrapMode) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	var err error
	*m, err = parseBootstrapMode(s)
	return err
}

// parseBootstrapMode reads a string value and returns a bootstrap mode.
func parseBootstrapMode(s string) (BootstrapMode, error) {
	switch strings.ToLower(s) {
	case "hosts":
		return BootstrapModeHosts, nil
	case "file":
		return BootstrapModeFile, nil
	}
	return BootstrapModeNone, errors.New("invalid or no ringpop bootstrap mode")
}

func validateBootstrapMode(rpConfig *Ringpop) error {
	switch rpConfig.BootstrapMode {
	case BootstrapModeFile:
		if len(rpConfig.BootstrapFile) == 0 {
			return fmt.Errorf("ringpop config missing bootstrap file param")
		}
	case BootstrapModeHosts:
		if len(rpConfig.BootstrapHosts) == 0 {
			return fmt.Errorf("ringpop config missing boostrap hosts param")
		}
	default:
		return fmt.Errorf("ringpop config with unknown boostrap mode")
	}
	return nil
}

func newRingpopFactory(rpConfig *Ringpop) (*RingpopFactory, error) {
	if err := rpConfig.validate(); err != nil {
		return nil, err
	}
	if rpConfig.MaxJoinDuration == 0 {
		rpConfig.MaxJoinDuration = defaultMaxJoinDuration
	}
	return &RingpopFactory{config: rpConfig}, nil
}

// CreateRingpop is the implementation for RingpopFactory.CreateRingpop
func (factory *RingpopFactory) CreateRingpop(ch *tchannel.Channel) (*ringpop.Ringpop, error) {

	discoveryProvider, err := newDiscoveryProvider(factory.config)
	if err != nil {
		return nil, err
	}

	rp, err := ringpop.New(factory.config.Name, ringpop.Channel(ch))
	if err != nil {
		return nil, err
	}

	bootstrapOpts := &swim.BootstrapOptions{
		MaxJoinDuration:  factory.config.MaxJoinDuration,
		DiscoverProvider: discoveryProvider,
	}

	_, err = rp.Bootstrap(bootstrapOpts)
	if err != nil {
		return nil, err
	}
	return rp, nil
}

func newDiscoveryProvider(cfg *Ringpop) (discovery.DiscoverProvider, error) {
	switch cfg.BootstrapMode {
	case BootstrapModeHosts:
		return statichosts.New(cfg.BootstrapHosts...), nil
	case BootstrapModeFile:
		return jsonfile.New(cfg.BootstrapFile), nil
	}
	return nil, fmt.Errorf("unknown bootstrap mode")
}