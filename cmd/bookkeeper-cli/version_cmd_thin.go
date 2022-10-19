//go:build !thick
// +build !thick

package main

func init() {
	// Add extra flags that we want only if building the thin CLI (which is the
	// default)
	versionCmdFlagSet.BoolP(
		flagInsecure,
		"k",
		false,
		"tolerate certificate errors for HTTPS connections",
	)
	versionCmdFlagSet.StringP(
		flagServer,
		"s",
		"",
		"specify the address of the Bookkeeper server (can also be set using "+
			"the BOOKKEEPER_SERVER environment variable)",
	)
}
