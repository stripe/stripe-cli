# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [commands.proto](#commands.proto)
    - [StripeCLI](#rpc.StripeCLI)
  
- [login.proto](#login.proto)
    - [LoginRequest](#rpc.LoginRequest)
    - [LoginResponse](#rpc.LoginResponse)
  
- [login_status.proto](#login_status.proto)
    - [LoginStatusRequest](#rpc.LoginStatusRequest)
    - [LoginStatusResponse](#rpc.LoginStatusResponse)
  
- [sample_configs.proto](#sample_configs.proto)
    - [SampleConfigsRequest](#rpc.SampleConfigsRequest)
    - [SampleConfigsResponse](#rpc.SampleConfigsResponse)
    - [SampleConfigsResponse.Integration](#rpc.SampleConfigsResponse.Integration)
  
- [sample_create.proto](#sample_create.proto)
    - [SampleCreateRequest](#rpc.SampleCreateRequest)
    - [SampleCreateResponse](#rpc.SampleCreateResponse)
  
- [samples_list.proto](#samples_list.proto)
    - [SamplesListRequest](#rpc.SamplesListRequest)
    - [SamplesListResponse](#rpc.SamplesListResponse)
    - [SamplesListResponse.SampleData](#rpc.SamplesListResponse.SampleData)
  
- [triggers_list.proto](#triggers_list.proto)
    - [TriggersListRequest](#rpc.TriggersListRequest)
    - [TriggersListResponse](#rpc.TriggersListResponse)
  
- [version.proto](#version.proto)
    - [VersionRequest](#rpc.VersionRequest)
    - [VersionResponse](#rpc.VersionResponse)
  
- [Scalar Value Types](#scalar-value-types)



<a name="commands.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## commands.proto


 

 

 


<a name="rpc.StripeCLI"></a>

### StripeCLI


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Login | [LoginRequest](#rpc.LoginRequest) | [LoginResponse](#rpc.LoginResponse) | Get a link to log in to the Stripe CLI. The client will have to open the browser to complete the login. Use `LoginStatus` after this method to wait for success. Like `stripe login`. |
| LoginStatus | [LoginStatusRequest](#rpc.LoginStatusRequest) | [LoginStatusResponse](#rpc.LoginStatusResponse) | Successfully returns when login has succeeded, or returns an error if login has failed or timed out. Use this method after `Login` to check for success. |
| SampleConfigs | [SampleConfigsRequest](#rpc.SampleConfigsRequest) | [SampleConfigsResponse](#rpc.SampleConfigsResponse) | Get a list of available configs for a given Stripe sample. |
| SampleCreate | [SampleCreateRequest](#rpc.SampleCreateRequest) | [SampleCreateResponse](#rpc.SampleCreateResponse) | Clone a Stripe sample. Like `stripe samples create`. |
| SamplesList | [SamplesListRequest](#rpc.SamplesListRequest) | [SamplesListResponse](#rpc.SamplesListResponse) | Get a list of available Stripe samples. Like `stripe samples list`. |
| TriggersList | [TriggersListRequest](#rpc.TriggersListRequest) | [TriggersListResponse](#rpc.TriggersListResponse) | Get a list of supported events for `Trigger`. |
| Version | [VersionRequest](#rpc.VersionRequest) | [VersionResponse](#rpc.VersionResponse) | Get the version of the Stripe CLI. Like `stripe version`. |

 



<a name="login.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## login.proto



<a name="rpc.LoginRequest"></a>

### LoginRequest







<a name="rpc.LoginResponse"></a>

### LoginResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The URL to complete the login. The client must open this in the browser to successfully log in. |
| pairing_code | [string](#string) |  | The pairing code to verify your authentication with Stripe, e.g. excels-champ-wins-quaint |





 

 

 

 



<a name="login_status.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## login_status.proto



<a name="rpc.LoginStatusRequest"></a>

### LoginStatusRequest







<a name="rpc.LoginStatusResponse"></a>

### LoginStatusResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account_id | [string](#string) |  | ID of the Stripe account, e.g. acct_123 |
| display_name | [string](#string) |  | Display name of the Stripe account |





 

 

 

 



<a name="sample_configs.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## sample_configs.proto



<a name="rpc.SampleConfigsRequest"></a>

### SampleConfigsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sample_name | [string](#string) |  | Name of the sample, e.g. accept-a-card-payment |






<a name="rpc.SampleConfigsResponse"></a>

### SampleConfigsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| integrations | [SampleConfigsResponse.Integration](#rpc.SampleConfigsResponse.Integration) | repeated | List of available integrations for this sample, e.g. the &#34;accept-a-card-payment&#34; sample includes an integration that uses webhooks, a web client, and a node server. |






<a name="rpc.SampleConfigsResponse.Integration"></a>

### SampleConfigsResponse.Integration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| integration_name | [string](#string) |  | Name of an available integration for this sample, e.g. &#34;using-webhooks&#34; |
| clients | [string](#string) | repeated | List of available languages or platforms for the sample client, e.g. [&#34;web&#34;, &#34;android&#34;, &#34;ios&#34;] |
| servers | [string](#string) | repeated | List of available languages or platforms for the sample server, e.g. [&#34;java&#34;, &#34;node&#34;] |





 

 

 

 



<a name="sample_create.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## sample_create.proto



<a name="rpc.SampleCreateRequest"></a>

### SampleCreateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sample_name | [string](#string) |  | Name of the sample, e.g. accept-a-card-payment. Use the `SamplesList` method to get a list of available samples. |
| integration_name | [string](#string) |  | Name of the particular integration, e.g. using-webhooks. Use the `SampleConfigs` method to get the available options. |
| client | [string](#string) |  | Platform or language for the client, e.g. web. Use the `SampleConfigs` method to get the available options. |
| server | [string](#string) |  | Platform or language for the server, e.g. node. Use the `SampleConfigs` method to get the available options. |
| path | [string](#string) |  | Path to clone the repo to. |
| force_refresh | [bool](#bool) |  | If true, clear the local cache before creating the sample. |






<a name="rpc.SampleCreateResponse"></a>

### SampleCreateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| post_install | [string](#string) |  | Additional instructions for the sample after install. |
| path | [string](#string) |  | Path to the sample. |





 

 

 

 



<a name="samples_list.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## samples_list.proto



<a name="rpc.SamplesListRequest"></a>

### SamplesListRequest







<a name="rpc.SamplesListResponse"></a>

### SamplesListResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| samples | [SamplesListResponse.SampleData](#rpc.SamplesListResponse.SampleData) | repeated | List of available Stripe samples |






<a name="rpc.SamplesListResponse.SampleData"></a>

### SamplesListResponse.SampleData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the sample, e.g. accept-a-card-payment |
| url | [string](#string) |  | URL of the repo, e.g. https://github.com/stripe-samples/accept-a-card-payment |
| description | [string](#string) |  | Description of the sample, e.g. Learn how to accept a basic card payment |





 

 

 

 



<a name="triggers_list.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## triggers_list.proto



<a name="rpc.TriggersListRequest"></a>

### TriggersListRequest







<a name="rpc.TriggersListResponse"></a>

### TriggersListResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| events | [string](#string) | repeated | A list of supported events for `Trigger`. |





 

 

 

 



<a name="version.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## version.proto



<a name="rpc.VersionRequest"></a>

### VersionRequest







<a name="rpc.VersionResponse"></a>

### VersionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | The version of the Stripe CLI |





 

 

 

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

