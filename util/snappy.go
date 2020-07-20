// Copyright 2020 tree xie
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

// snappy compress

package util

import (
	"bytes"
	"io/ioutil"

	"github.com/golang/snappy"
)

func DecodeSnappy(buf []byte) ([]byte, error) {
	// 如果是0xff，表示通过new writer的形式压缩
	if buf[0] == 0xff {
		r := snappy.NewReader(bytes.NewReader(buf))
		return ioutil.ReadAll(r)
	}
	var dst []byte
	return snappy.Decode(dst, buf)
}
