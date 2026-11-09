module github.com/akuity/kargo/pkg/x/client/watch

go 1.27.0

replace github.com/akuity/kargo/pkg/x/client/generated => ../generated

require (
	github.com/akuity/kargo/pkg/x/client/generated v0.0.0
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/stretchr/testify v1.12.1
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect
