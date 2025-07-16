/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * The MIT License (MIT)
 *
 * Copyright (c) 2014 Manuel Martínez-Almeida
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package sse

import (
	"bytes"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/common/json"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/cloudwego/hertz/pkg/network"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/http1/resp"
)

type myStruct struct {
	A int
	B string `json:"value"`
}

func unsafeMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestEncode(t *testing.T) {
	test := []struct {
		Name    string
		Event   *Event
		WantErr byte
		Want    string
	}{
		{
			Name: "with data only",
			Want: `data:junk
data:
data:jk
data:id:fake

`,
			Event: &Event{
				Data: []byte("junk\n\njk\nid:fake"),
			},
		},
		{
			Name: "with event",
			Want: `event:t\n:<>\r	est
data:junk
data:
data:jk
data:id:fake

`,
			Event: &Event{
				Event: "t\n:<>\r\test",
				Data:  []byte("junk\n\njk\nid:fake"),
			},
		},
		{
			Name: "with id",
			Event: &Event{
				ID:   "t\n:<>\r\test",
				Data: []byte("junk\n\njk\nid:fa\rke"),
			},
			Want: `id:t\n:<>\r	est
data:junk
data:
data:jk
data:id:fa\rke

`,
		},
		{
			Name: "with retry",
			Event: &Event{
				Retry: 11,
				Data:  []byte("junk\n\njk\nid:fake\n"),
			},
			Want: `retry:11
data:junk
data:
data:jk
data:id:fake
data:

`,
		},
		{
			Name: "with everything",
			Event: &Event{
				Event: "abc",
				ID:    "12345",
				Retry: 10,
				Data:  []byte("some data"),
			},
			Want: "id:12345\nevent:abc\nretry:10\ndata:some data\n\n",
		},
		{
			Name: "encode map",
			Event: &Event{
				Event: "a slice",
				Data:  unsafeMarshal(myStruct{1, "number"}),
			},
			Want: "event:a slice\ndata:{\"A\":1,\"value\":\"number\"}\n\n",
		},
		{
			Name: "encode struct",
			Event: &Event{
				Event: "a struct",
				Data:  unsafeMarshal(myStruct{1, "number"}),
			},
			Want: "event:a struct\ndata:{\"A\":1,\"value\":\"number\"}\n\n",
		},
		{
			Name: "struct pointer",
			Event: &Event{
				Event: "a struct",
				Data:  unsafeMarshal(&myStruct{1, "number"}),
			},
			Want: "event:a struct\ndata:{\"A\":1,\"value\":\"number\"}\n\n",
		},
		{
			Name: "encode integer",
			Event: &Event{
				Event: "an integer",
				Data:  []byte("1"),
			},
			Want: "event:an integer\ndata:1\n\n",
		},
		{
			Name: "encode float",
			Event: &Event{
				Event: "Float",
				Data:  []byte("1.5"),
			},
			Want: "event:Float\ndata:1.5\n\n",
		},
		{
			Name: "encode string",
			Event: &Event{
				Event: "String",
				Data:  []byte("hertz"),
			},
			Want: "event:String\ndata:hertz\n\n",
		},
	}
	for _, tt := range test {
		var b bytes.Buffer
		err := Encode(&b, tt.Event)
		got := b.String()
		assert.Nil(t, err)
		assert.DeepEqual(t, got, tt.Want)
	}
}

func TestEncodeStream(t *testing.T) {
	w := new(bytes.Buffer)
	event := &Event{
		Event: "float",
		Data:  []byte("1.5"),
	}
	err := Encode(w, event)
	assert.Nil(t, err)

	event = &Event{
		ID:   "123",
		Data: unsafeMarshal(map[string]interface{}{"foo": "bar", "bar": "foo"}),
	}
	err = Encode(w, event)
	assert.Nil(t, err)

	event = &Event{
		ID:    "124",
		Event: "chat",
		Data:  []byte("hi! dude"),
	}
	err = Encode(w, event)
	assert.Nil(t, err)
	assert.DeepEqual(t, "event:float\ndata:1.5\n\nid:123\ndata:{\"bar\":\"foo\",\"foo\":\"bar\"}\n\nid:124\nevent:chat\ndata:hi! dude\n\n", w.String())
}

func BenchmarkFullSSE(b *testing.B) {
	buf := new(bytes.Buffer)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Encode(buf, &Event{
			Event: "new_message",
			ID:    "13435",
			Retry: 10,
			Data:  []byte("hi! how are you? I am fine. this is a long stupid message!!!"),
		})
		buf.Reset()
	}
}

func BenchmarkNoRetrySSE(b *testing.B) {
	buf := new(bytes.Buffer)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Encode(buf, &Event{
			Event: "new_message",
			ID:    "13435",
			Data:  []byte("hi! how are you? I am fine. this is a long stupid message!!!"),
		})
		buf.Reset()
	}
}

func BenchmarkSimpleSSE(b *testing.B) {
	buf := new(bytes.Buffer)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		time.Sleep(time.Second)
		_ = Encode(buf, &Event{
			Event: "new_message",
			Data:  []byte("hi! how are you? I am fine. this is a long stupid message!!!"),
		})
		buf.Reset()
	}
}

func TestEncodeWithChunkedBodyWriter(t *testing.T) {
	mw := &MockWriter{}

	w := resp.NewChunkedBodyWriter(&protocol.Response{}, mw)
	// Set discard and call Write to skip writing HTTP headers.
	mw.SetDiscard(true)
	_, _ = w.Write(nil)
	mw.SetDiscard(false)

	_ = Encode(w, &Event{
		Event: "message",
		ID:    "1",
		Retry: 2,
		Data:  []byte("test data"),
	})

	assert.DeepEqual(t, "\r\nid:1\nevent:message\nretry:2\ndata:test data\n\n\r\n", string(mw.Bytes()))
}

type MockWriter struct {
	discard bool
	buffer  bytes.Buffer
}

var _ network.Writer = (*MockWriter)(nil)

func (w *MockWriter) Malloc(n int) (buf []byte, err error) {
	return nil, nil
}

func (w *MockWriter) WriteBinary(b []byte) (n int, err error) {
	if w.discard {
		return len(b), nil
	}
	return w.buffer.Write(b)
}

func (w *MockWriter) Flush() error {
	return nil
}

func (w *MockWriter) Bytes() []byte {
	return w.buffer.Bytes()
}

func (w *MockWriter) SetDiscard(discard bool) {
	w.discard = discard
}
