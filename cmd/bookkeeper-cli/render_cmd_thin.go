//go:build !thick
// +build !thick

package main

func init() {
	// Add extra flags that we want only if building the thin CLI (which is the
	// default)
	renderCmdFlagSet.BoolP(
		flagInsecure,
		"k",
		false,
		"tolerate certificate errors for HTTPS connections",
	)
	renderCmdFlagSet.StringP(
		flagServer,
		"s",
		"",
		"specify the address of the Bookkeeper server (required; can also be "+
			"set using the BOOKKEEPER_SERVER environment variable)",
	)
}
