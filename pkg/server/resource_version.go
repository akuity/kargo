package server

import "strconv"

// effectiveResourceVersion returns rv if it is a real, non-default Kubernetes
// resource version (non-empty and non-"0"). Otherwise it computes the maximum
// resource version from the provided item versions, which serves as a safe
// lower bound for a subsequent Watch: any event with a higher resource version
// is guaranteed to be newer than the listed state.
//
// Background: the controller-runtime cached client returns "0" for the
// list-level ResourceVersion, which causes a Kubernetes Watch to replay all
// existing objects as ADDED events. Using the maximum object-level resource
// version avoids this replay while still being correct: every modification to
// a listed object, and every new object created after the list, will have a
// resource version greater than this value.
func effectiveResourceVersion(rv string, itemVersions []string) string {
	if rv != "" && rv != "0" {
		return rv
	}
	var maxRV int64
	for _, v := range itemVersions {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			continue
		}
		if n > maxRV {
			maxRV = n
		}
	}
	if maxRV == 0 {
		return ""
	}
	return strconv.FormatInt(maxRV, 10)
}
