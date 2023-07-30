package protocol

import (
	"encoding/json"
	"testing"
)

func TestRequestNotification(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		expected bool
	}{
		{
			name: "messages without an ID are notifications",
			msg: `{
	"jsonrpc": "2.0",
	"method": "notification",
	"params": null
}`,
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var msg Request
			err := json.Unmarshal([]byte(test.msg), &msg)
			if err != nil {
				t.Fatalf("failed to unmarshal message: %v", err)
			}
			actual := msg.IsNotification()
			if test.expected != actual {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}
