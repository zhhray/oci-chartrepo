# ScannerRegistration

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Uuid** | **string** | The unique identifier of this registration. | [optional] [default to null]
**Name** | **string** | The name of this registration. | [optional] [default to null]
**Description** | **string** | An optional description of this registration. | [optional] [default to null]
**Url** | **string** | A base URL of the scanner adapter | [optional] [default to null]
**Disabled** | **bool** | Indicate whether the registration is enabled or not | [optional] [default to null]
**IsDefault** | **bool** | Indicate if the registration is set as the system default one | [optional] [default to null]
**Health** | **string** | Indicate the healthy of the registration | [optional] [default to null]
**Auth** | **string** | Specify what authentication approach is adopted for the HTTP communications. Supported types Basic\&quot;, \&quot;Bearer\&quot; and api key header \&quot;X-ScannerAdapter-API-Key\&quot;  | [optional] [default to null]
**AccessCredential** | **string** | An optional value of the HTTP Authorization header sent with each request to the Scanner Adapter API.  | [optional] [default to null]
**SkipCertVerify** | **bool** | Indicate if skip the certificate verification when sending HTTP requests | [optional] [default to null]
**UseInternalAddr** | **bool** | Indicate whether use internal registry addr for the scanner to pull content or not | [optional] [default to null]
**Adapter** | **string** | Optional property to describe the name of the scanner registration | [optional] [default to null]
**Vendor** | **string** | Optional property to describe the vendor of the scanner registration | [optional] [default to null]
**Version** | **string** | Optional property to describe the version of the scanner registration | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


