# DiscoveredRef

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** | ID is the identifier of the object the ref points to, typically a SHA-1 hash. For an annotated tag this is the tag object&#39;s ID, not the commit it dereferences to, because the value is obtained via git ls-remote --refs. This is immaterial to its sole use -- change detection -- since the value moves whenever the ref is re-pointed and is only ever compared against other values obtained the same way.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 
**Name** | Pointer to **string** | Name is the short name of the ref (e.g. a tag name such as \&quot;v1.2.3\&quot;), without its \&quot;refs/tags/\&quot; or \&quot;refs/heads/\&quot; prefix.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 

## Methods

### NewDiscoveredRef

`func NewDiscoveredRef() *DiscoveredRef`

NewDiscoveredRef instantiates a new DiscoveredRef object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDiscoveredRefWithDefaults

`func NewDiscoveredRefWithDefaults() *DiscoveredRef`

NewDiscoveredRefWithDefaults instantiates a new DiscoveredRef object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *DiscoveredRef) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *DiscoveredRef) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *DiscoveredRef) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *DiscoveredRef) HasId() bool`

HasId returns a boolean if a field has been set.

### GetName

`func (o *DiscoveredRef) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *DiscoveredRef) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *DiscoveredRef) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *DiscoveredRef) HasName() bool`

HasName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


