package common

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasPreferDirective(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   bool
	}{
		{name: "exact", values: []string{"respond-async"}, want: true},
		{name: "case insensitive with parameters", values: []string{"RESPOND-ASYNC; handling=strict"}, want: true},
		{name: "comma separated", values: []string{"return=minimal, respond-async"}, want: true},
		{name: "repeated fields", values: []string{"return=minimal", "respond-async"}, want: true},
		{name: "quoted comma does not split", values: []string{`example="a,b", respond-async`}, want: true},
		{name: "substring rejected", values: []string{"x-respond-async"}},
		{name: "quoted value rejected", values: []string{`example="respond-async"`}},
		{name: "valued form rejected", values: []string{"respond-async=false"}},
		{name: "missing", values: []string{"return=minimal"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := make(http.Header)
			for _, value := range tt.values {
				header.Add("Prefer", value)
			}
			assert.Equal(t, tt.want, HasPreferDirective(header, "respond-async"))
		})
	}
}
