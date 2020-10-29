# \SystemApi

All URIs are relative to *http://localhost/api/v2.0*

Method | HTTP request | Description
------------- | ------------- | -------------
[**SystemCVEAllowlistGet**](SystemApi.md#SystemCVEAllowlistGet) | **Get** /system/CVEAllowlist | Get the system level allowlist of CVE.
[**SystemCVEAllowlistPut**](SystemApi.md#SystemCVEAllowlistPut) | **Put** /system/CVEAllowlist | Update the system level allowlist of CVE.
[**SystemOidcPingPost**](SystemApi.md#SystemOidcPingPost) | **Post** /system/oidc/ping | Test the OIDC endpoint.


# **SystemCVEAllowlistGet**
> CveAllowlist SystemCVEAllowlistGet(ctx, )
Get the system level allowlist of CVE.

Get the system level allowlist of CVE.  This API can be called by all authenticated users.

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**CveAllowlist**](CVEAllowlist.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, text/plain

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **SystemCVEAllowlistPut**
> SystemCVEAllowlistPut(ctx, optional)
Update the system level allowlist of CVE.

This API overwrites the system level allowlist of CVE with the list in request body.  Only system Admin has permission to call this API.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***SystemApiSystemCVEAllowlistPutOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a SystemApiSystemCVEAllowlistPutOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **allowlist** | [**optional.Interface of CveAllowlist**](CveAllowlist.md)| The allowlist with new content | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, text/plain

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **SystemOidcPingPost**
> SystemOidcPingPost(ctx, endpoint)
Test the OIDC endpoint.

Test the OIDC endpoint, the setting of the endpoint is provided in the request.  This API can only be called by system admin.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **endpoint** | [**Endpoint**](Endpoint.md)| Request body for OIDC endpoint to be tested. | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, text/plain

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

