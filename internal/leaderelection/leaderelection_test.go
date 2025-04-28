package leaderelection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		additions []string
		want      string
	}{
		{
			name: "empty id",
			id:   "",
			want: "",
		},
		{
			name:      "empty id with additions",
			id:        "",
			additions: []string{"foo", "bar"},
			want:      "",
		},
		{
			name: "id without additions",
			id:   "foo",
			want: "foo",
		},
		{
			name:      "id with additions",
			id:        "foo",
			additions: []string{"bar", "baz"},
			want:      "foo-c8f8b724",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GenerateID(tt.id, tt.additions...))
		})
	}
}

func TestGenerateIDFromLabelSelector(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		selector labels.Selector
		want     string
	}{
		{
			name: "empty id and selector",
			id:   "",
			want: "",
		},
		{
			name:     "empty id with selector",
			id:       "",
			selector: labels.Set{"foo": "bar"}.AsSelector(),
			want:     "",
		},
		{
			name:     "id without selector",
			id:       "foo",
			selector: nil,
			want:     "foo",
		},
		{
			name:     "id with empty selector",
			id:       "foo",
			selector: labels.Set{}.AsSelector(),
			want:     "foo",
		},
		{
			name:     "id with selector",
			id:       "foo",
			selector: labels.Set{"foo": "bar"}.AsSelector(),
			want:     "foo-3ba8907e",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GenerateIDFromLabelSelector(tt.id, tt.selector))
		})
	}
}
