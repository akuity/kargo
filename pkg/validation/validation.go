package validation

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func MinLength(f *field.Path, value string, minLength int) *field.Error {
	if len(value) < minLength {
		return field.Invalid(
			f,
			value,
			fmt.Sprintf("must have a minimum length of %d", minLength),
		)
	}
	return nil
}

func MaxLength(f *field.Path, value string, maxLength int) *field.Error {
	if len(value) > maxLength {
		return field.Invalid(
			f,
			value,
			fmt.Sprintf("must have a maximum length of %d", maxLength),
		)
	}
	return nil
}

func SemverConstraint(f *field.Path, semverConstraint string) *field.Error {
	if semverConstraint == "" {
		return nil
	}
	if _, err := semver.NewConstraint(semverConstraint); err != nil {
		return field.Invalid(f, semverConstraint, "")
	}
	return nil
}
