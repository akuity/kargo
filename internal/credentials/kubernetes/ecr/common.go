package ecr

import "regexp"

var ecrURLRegex = regexp.MustCompile(`^(oci://)?[0-9]{12}\.dkr\.ecr\.(.+)\.amazonaws\.com/`)
