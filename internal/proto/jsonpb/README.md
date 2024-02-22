# jsonpb

`jsonpb` is a fork of the [gogo/protobuf](https://github.com/gogo/protobuf)'s `jsonpb` package 
to support unmarshaling Kubernetes types.

Some Kubernetes types (e.g. `metav1.Time`) are not compatible with the standard `jsonpb` package
since the protobuf support added later. ([related issue](https://github.com/kubernetes/apimachinery/issues/59#issuecomment-449257201)).

To overcome this limitation, this fork patches `Unmarshaler` to check if the given type implements 
`json.Unmarshaler` interface and unmarshal data with it.
