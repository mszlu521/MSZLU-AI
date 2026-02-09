/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package store

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/compose"
	"github.com/mszlu521/thunder/logs"
)

func NewInMemoryStore() compose.CheckPointStore {
	return &inMemoryStore{
		mem: map[string][]byte{},
	}
}

type inMemoryStore struct {
	mem  map[string][]byte
	lock sync.RWMutex
}

func (i *inMemoryStore) Set(ctx context.Context, key string, value []byte) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.mem[key] = value
	logs.Infof("set key: %s, value: %s", key, value)
	return nil
}

func (i *inMemoryStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	v, ok := i.mem[key]
	logs.Infof("get key: %s, value: %s", key, v)
	return v, ok, nil
}
