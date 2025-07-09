package external

import "testing"

func Test_needsRefresh_Git(t *testing.T) {
	for _, test := range []struct{
		name string
	}{
		{
			name: "semver selection",
		},
	}{
		t.Run(test.name, func(t *testing.T) {
		})
	}
}

func Test_needsRefresh_Image(t *testing.T) {
	for _, test := range []struct{
		name string
	}{
		{
			name: "test case 1",
		},
	}{
		t.Run(test.name, func(t *testing.T) {
		})
	}
}

func Test_needsRefresh_Chart(t *testing.T) {
	for _, test := range []struct{
		name string
	}{
		{
			name: "test case 1",
		},
	}{
		t.Run(test.name, func(t *testing.T) {
		})
	}
}