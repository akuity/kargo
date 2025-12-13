package image

import "time"

type platform struct {
	OS          string            `json:"os,omitempty"`
	Arch        string            `json:"arch,omitempty"`
	Variant     string            `json:"variant,omitempty"`
	CreatedAt   *time.Time        `json:"createdAt,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
