// Copyright 2019 tree xie
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

// location config

package config

import "sort"

// Location location config
type Location struct {
	Name           string   `yaml:"name,omitempty" json:"name,omitempty"`
	Upstream       string   `yaml:"upstream,omitempty" json:"upstream,omitempty"`
	Server         string   `yaml:"server,omitempty" json:"server,omitempty"`
	Cache          string   `yaml:"cache,omitempty" json:"cache,omitempty"`
	Prefixs        []string `yaml:"prefixs,omitempty" json:"prefixs,omitempty"`
	Rewrites       []string `yaml:"rewrites,omitempty" json:"rewrites,omitempty"`
	Hosts          []string `yaml:"hosts,omitempty" json:"hosts,omitempty"`
	ResponseHeader []string `yaml:"responseHeader,omitempty" json:"responseHeader,omitempty"`
	RequestHeader  []string `yaml:"requestHeader,omitempty" json:"requestHeader,omitempty"`
}

// Locations locations
type Locations []*Location

// Fetch fetch location config
func (l *Location) Fetch() (err error) {
	return fetchConfig(l, defaultLocationPath, l.Name)
}

// Save save location config
func (l *Location) Save() (err error) {
	return saveConfig(l, defaultLocationPath, l.Name)
}

// Delete delete location config
func (l *Location) Delete() (err error) {
	return deleteConfig(defaultLocationPath, l.Name)
}

func (l *Location) getPriority() int {
	priority := 8
	if len(l.Prefixs) != 0 {
		priority -= 4
	}
	if len(l.Hosts) != 0 {
		priority -= 2
	}
	return priority
}

// Sort sort locations
func (locations Locations) Sort() {
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].getPriority() < locations[j].getPriority()
	})
}

// Get get location config from location list
func (locations Locations) Get(name string) (l *Location) {
	for _, item := range locations {
		if item.Name == name {
			l = item
		}
	}
	return
}

// GetLocations get locations
// *Location for better performance)
func GetLocations() (locations Locations, err error) {
	keys, err := listKeysExcludePrefix(defaultLocationPath)
	if err != nil {
		return
	}
	locations = make(Locations, 0, len(keys))
	for _, key := range keys {
		l := &Location{
			Name: key,
		}
		err = l.Fetch()
		if err != nil {
			return
		}
		locations = append(locations, l)
	}
	return
}
