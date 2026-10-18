// security-fixture-server is an isolated, synthetic upstream for gateway acceptance.
package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

func main() {
	var mu sync.Mutex
	requests := make([]map[string]interface{}, 0)
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		data, _ := common.Marshal(requests)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})
	http.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"object":"list","data":[{"id":"security-chat","object":"model"},{"id":"security-image","object":"model"}]}`)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := common.DecodeJson(http.MaxBytesReader(w, r.Body, 1<<20), &body); err != nil {
			http.Error(w, "invalid fixture input", 400)
			return
		}
		mu.Lock()
		requests = append(requests, map[string]interface{}{"path": r.URL.Path, "body": body})
		mu.Unlock()
		if r.URL.Path == "/v1/images/generations" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"created":1,"data":[{"b64_json":"aXNvbGF0ZWQtdGVzdA=="}]}`)
			return
		}
		if body["stream"] == true {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"fixture-ok\"},\"finish_reason\":null}]}\n\n")
			_, _ = fmt.Fprint(w, "data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":2,\"total_tokens\":12}}\n\ndata: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":"fixture","object":"chat.completion","model":"security-chat","choices":[{"index":0,"message":{"role":"assistant","content":"fixture-ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`)
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
