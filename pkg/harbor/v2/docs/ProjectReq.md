# ProjectReq

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ProjectName** | **string** | The name of the project. | [optional] [default to null]
**Public** | **bool** | deprecated, reserved for project creation in replication | [optional] [default to null]
**Metadata** | [***ProjectMetadata**](ProjectMetadata.md) | The metadata of the project. | [optional] [default to null]
**CveAllowlist** | [***CveAllowlist**](CVEAllowlist.md) | The CVE allowlist of the project. | [optional] [default to null]
**StorageLimit** | **int64** | The storage quota of the project. | [optional] [default to null]
**RegistryId** | **int64** | The ID of referenced registry when creating the proxy cache project | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


