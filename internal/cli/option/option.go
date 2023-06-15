package option

import (
	"github.com/bufbuild/connect-go"
)

type Option struct {
	ServerURL      string
	UseLocalServer bool

	ClientOption connect.ClientOption
}
