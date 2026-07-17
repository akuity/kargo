# V1PolicyRule

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApiGroups** | Pointer to **[]string** | APIGroups is the name of the APIGroup that contains the resources.  If multiple API groups are specified, any action requested against one of the enumerated resources in any API group will be allowed. \&quot;\&quot; represents the core API group and \&quot;*\&quot; represents all API groups. +optional +listType&#x3D;atomic | [optional] 
**NonResourceURLs** | Pointer to **[]string** | NonResourceURLs is a set of partial urls that a user should have access to.  *s are allowed, but only as the full, final step in the path Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding. Rules can either apply to API resources (such as \&quot;pods\&quot; or \&quot;secrets\&quot;) or non-resource URL paths (such as \&quot;/api\&quot;),  but not both. +optional +listType&#x3D;atomic | [optional] 
**ResourceNames** | Pointer to **[]string** | ResourceNames is an optional white list of names that the rule applies to.  An empty set means that everything is allowed. +optional +listType&#x3D;atomic | [optional] 
**Resources** | Pointer to **[]string** | Resources is a list of resources this rule applies to. &#39;*&#39; represents all resources. +optional +listType&#x3D;atomic | [optional] 
**Verbs** | Pointer to **[]string** | Verbs is a list of Verbs that apply to ALL the ResourceKinds contained in this rule. &#39;*&#39; represents all verbs. +listType&#x3D;atomic | [optional] 

## Methods

### NewV1PolicyRule

`func NewV1PolicyRule() *V1PolicyRule`

NewV1PolicyRule instantiates a new V1PolicyRule object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PolicyRuleWithDefaults

`func NewV1PolicyRuleWithDefaults() *V1PolicyRule`

NewV1PolicyRuleWithDefaults instantiates a new V1PolicyRule object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApiGroups

`func (o *V1PolicyRule) GetApiGroups() []string`

GetApiGroups returns the ApiGroups field if non-nil, zero value otherwise.

### GetApiGroupsOk

`func (o *V1PolicyRule) GetApiGroupsOk() (*[]string, bool)`

GetApiGroupsOk returns a tuple with the ApiGroups field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiGroups

`func (o *V1PolicyRule) SetApiGroups(v []string)`

SetApiGroups sets ApiGroups field to given value.

### HasApiGroups

`func (o *V1PolicyRule) HasApiGroups() bool`

HasApiGroups returns a boolean if a field has been set.

### GetNonResourceURLs

`func (o *V1PolicyRule) GetNonResourceURLs() []string`

GetNonResourceURLs returns the NonResourceURLs field if non-nil, zero value otherwise.

### GetNonResourceURLsOk

`func (o *V1PolicyRule) GetNonResourceURLsOk() (*[]string, bool)`

GetNonResourceURLsOk returns a tuple with the NonResourceURLs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNonResourceURLs

`func (o *V1PolicyRule) SetNonResourceURLs(v []string)`

SetNonResourceURLs sets NonResourceURLs field to given value.

### HasNonResourceURLs

`func (o *V1PolicyRule) HasNonResourceURLs() bool`

HasNonResourceURLs returns a boolean if a field has been set.

### GetResourceNames

`func (o *V1PolicyRule) GetResourceNames() []string`

GetResourceNames returns the ResourceNames field if non-nil, zero value otherwise.

### GetResourceNamesOk

`func (o *V1PolicyRule) GetResourceNamesOk() (*[]string, bool)`

GetResourceNamesOk returns a tuple with the ResourceNames field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceNames

`func (o *V1PolicyRule) SetResourceNames(v []string)`

SetResourceNames sets ResourceNames field to given value.

### HasResourceNames

`func (o *V1PolicyRule) HasResourceNames() bool`

HasResourceNames returns a boolean if a field has been set.

### GetResources

`func (o *V1PolicyRule) GetResources() []string`

GetResources returns the Resources field if non-nil, zero value otherwise.

### GetResourcesOk

`func (o *V1PolicyRule) GetResourcesOk() (*[]string, bool)`

GetResourcesOk returns a tuple with the Resources field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResources

`func (o *V1PolicyRule) SetResources(v []string)`

SetResources sets Resources field to given value.

### HasResources

`func (o *V1PolicyRule) HasResources() bool`

HasResources returns a boolean if a field has been set.

### GetVerbs

`func (o *V1PolicyRule) GetVerbs() []string`

GetVerbs returns the Verbs field if non-nil, zero value otherwise.

### GetVerbsOk

`func (o *V1PolicyRule) GetVerbsOk() (*[]string, bool)`

GetVerbsOk returns a tuple with the Verbs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVerbs

`func (o *V1PolicyRule) SetVerbs(v []string)`

SetVerbs sets Verbs field to given value.

### HasVerbs

`func (o *V1PolicyRule) HasVerbs() bool`

HasVerbs returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


