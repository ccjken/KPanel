package main

import (
	"encoding/json"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"strings"
	"testing"
	"time"
)

func TestDecodeNodeCheckStatusRejectsOversizeUnknownAndTrailingData(t *testing.T) {
	now := time.Now().UTC()
	value := contract.ServiceCheckSummary{Epoch: strings.Repeat("a", 32), IntervalSeconds: 300, Available: true, Items: []contract.ServiceCheckStatus{}}
	body, _ := json.Marshal(value)
	if decodeNodeCheckStatus(body, now) == nil {
		t.Fatal("valid summary rejected")
	}
	for _, data := range [][]byte{append(append([]byte{}, body...), []byte(" {}")...), []byte(strings.Repeat("x", contract.MaxServiceCheckSummaryBytes+1)), []byte(strings.Replace(string(body), "\"epoch\"", "\"secret\"", 1))} {
		if decodeNodeCheckStatus(data, now) != nil {
			t.Fatal("invalid summary accepted")
		}
	}
}
