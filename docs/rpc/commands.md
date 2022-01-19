# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [commands.proto](#commands.proto)
    - [StripeCLI](#rpc.StripeCLI)
  
- [common.proto](#common.proto)
    - [StripeEvent](#rpc.StripeEvent)
    - [StripeEvent.Request](#rpc.StripeEvent.Request)
  
- [events_resend.proto](#events_resend.proto)
    - [EventsResendRequest](#rpc.EventsResendRequest)
    - [EventsResendResponse](#rpc.EventsResendResponse)
  
- [fixtures.proto](#fixtures.proto)
    - [FixtureRequest](#rpc.FixtureRequest)
    - [FixtureResponse](#rpc.FixtureResponse)
  
- [listen.proto](#listen.proto)
    - [ListenRequest](#rpc.ListenRequest)
    - [ListenResponse](#rpc.ListenResponse)
    - [ListenResponse.EndpointResponse](#rpc.ListenResponse.EndpointResponse)
    - [ListenResponse.EndpointResponse.Data](#rpc.ListenResponse.EndpointResponse.Data)
  
    - [ListenResponse.EndpointResponse.Data.HttpMethod](#rpc.ListenResponse.EndpointResponse.Data.HttpMethod)
    - [ListenResponse.State](#rpc.ListenResponse.State)
  
- [login.proto](#login.proto)
    - [LoginRequest](#rpc.LoginRequest)
    - [LoginResponse](#rpc.LoginResponse)
  
- [login_status.proto](#login_status.proto)
    - [LoginStatusRequest](#rpc.LoginStatusRequest)
    - [LoginStatusResponse](#rpc.LoginStatusResponse)
  
- [logs_tail.proto](#logs_tail.proto)
    - [LogsTailRequest](#rpc.LogsTailRequest)
    - [LogsTailResponse](#rpc.LogsTailResponse)
    - [LogsTailResponse.Log](#rpc.LogsTailResponse.Log)
    - [LogsTailResponse.Log.Error](#rpc.LogsTailResponse.Log.Error)
  
    - [LogsTailRequest.Account](#rpc.LogsTailRequest.Account)
    - [LogsTailRequest.HttpMethod](#rpc.LogsTailRequest.HttpMethod)
    - [LogsTailRequest.RequestStatus](#rpc.LogsTailRequest.RequestStatus)
    - [LogsTailRequest.Source](#rpc.LogsTailRequest.Source)
    - [LogsTailRequest.StatusCodeType](#rpc.LogsTailRequest.StatusCodeType)
    - [LogsTailResponse.State](#rpc.LogsTailResponse.State)
  
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
  
- [trigger.proto](#trigger.proto)
    - [TriggerRequest](#rpc.TriggerRequest)
    - [TriggerResponse](#rpc.TriggerResponse)
  
- [triggers_list.proto](#triggers_list.proto)
    - [TriggersListRequest](#rpc.TriggersListRequest)
    - [TriggersListResponse](#rpc.TriggersListResponse)
  
- [version.proto](#version.proto)
    - [VersionRequest](#rpc.VersionRequest)
    - [VersionResponse](#rpc.VersionResponse)
  
- [webhook_endpoints_list.proto](#webhook_endpoints_list.proto)
    - [WebhookEndpointsListRequest](#rpc.WebhookEndpointsListRequest)
    - [WebhookEndpointsListResponse](#rpc.WebhookEndpointsListResponse)
    - [WebhookEndpointsListResponse.WebhookEndpointData](#rpc.WebhookEndpointsListResponse.WebhookEndpointData)
  
- [Scalar Value Types](#scalar-value-types)



<a name="commands.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## commands.proto


 

 

 


<a name="rpc.StripeCLI"></a>

### StripeCLI


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| EventsResend | [EventsResendRequest](#rpc.EventsResendRequest) | [EventsResendResponse](#rpc.EventsResendResponse) | Resend an event given an event ID. Like `stripe events resend`. |
| Fixture | [FixtureRequest](#rpc.FixtureRequest) | [FixtureResponse](#rpc.FixtureResponse) | Retrieve the default fixture of given triggering event. |
| Listen | [ListenRequest](#rpc.ListenRequest) | [ListenResponse](#rpc.ListenResponse) stream | Receive webhook events from the Stripe API to your local machine. Like `stripe listen`. |
| Login | [LoginRequest](#rpc.LoginRequest) | [LoginResponse](#rpc.LoginResponse) | Get a link to log in to the Stripe CLI. The client will have to open the browser to complete the login. Use `LoginStatus` after this method to wait for success. Like `stripe login`. |
| LoginStatus | [LoginStatusRequest](#rpc.LoginStatusRequest) | [LoginStatusResponse](#rpc.LoginStatusResponse) | Successfully returns when login has succeeded, or returns an error if login has failed or timed out. Use this method after `Login` to check for success. |
| LogsTail | [LogsTailRequest](#rpc.LogsTailRequest) | [LogsTailResponse](#rpc.LogsTailResponse) stream | Get a realtime stream of API logs. Like `stripe logs tail`. |
| SampleConfigs | [SampleConfigsRequest](#rpc.SampleConfigsRequest) | [SampleConfigsResponse](#rpc.SampleConfigsResponse) | Get a list of available configs for a given Stripe sample. |
| SampleCreate | [SampleCreateRequest](#rpc.SampleCreateRequest) | [SampleCreateResponse](#rpc.SampleCreateResponse) | Clone a Stripe sample. Like `stripe samples create`. |
| SamplesList | [SamplesListRequest](#rpc.SamplesListRequest) | [SamplesListResponse](#rpc.SamplesListResponse) | Get a list of available Stripe samples. Like `stripe samples list`. |
| Trigger | [TriggerRequest](#rpc.TriggerRequest) | [TriggerResponse](#rpc.TriggerResponse) | Trigger a webhook event. Like `stripe trigger`. |
| TriggersList | [TriggersListRequest](#rpc.TriggersListRequest) | [TriggersListResponse](#rpc.TriggersListResponse) | Get a list of supported events for `Trigger`. |
| Version | [VersionRequest](#rpc.VersionRequest) | [VersionResponse](#rpc.VersionResponse) | Get the version of the Stripe CLI. Like `stripe version`. |
| WebhookEndpointsList | [WebhookEndpointsListRequest](#rpc.WebhookEndpointsListRequest) | [WebhookEndpointsListResponse](#rpc.WebhookEndpointsListResponse) | Get the list of webhook endpoints. |

 



<a name="common.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## common.proto



<a name="rpc.StripeEvent"></a>

### StripeEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique identifier for the object. |
| api_version | [string](#string) |  | The Stripe API version used to render `data`. Note: This property is populated only for events on or after October 31, 2014. |
| data | [google.protobuf.Struct](#google.protobuf.Struct) |  | Object containing data associated with the event. |
| request | [StripeEvent.Request](#rpc.StripeEvent.Request) |  | Information on the API request that instigated the event. |
| type | [string](#string) |  | Description of the event (e.g., invoice.created or charge.refunded). |
| account | [string](#string) |  | CONNECT ONLY* The connected account that originated the event. |
| created | [int64](#int64) |  | Time at which the object was created. Measured in seconds since the Unix epoch. |
| livemode | [bool](#bool) |  | Has the value true if the object exists in live mode or the value false if the object exists in test mode. |
| pending_webhooks | [int64](#int64) |  | Number of webhooks that have yet to be successfully delivered (i.e., to return a 20x response) to the URLs you’ve specified. |






<a name="rpc.StripeEvent.Request"></a>

### StripeEvent.Request



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | ID of the API request that caused the event. If null, the event was automatic (e.g., Stripe’s automatic subscription handling). Request logs are available in the dashboard, but currently not in the API. |
| idempotency_key | [string](#string) |  | The idempotency key transmitted during the request, if any. Note: This property is populated only for events on or after May 23, 2017. |





 

 

 

 



<a name="events_resend.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## events_resend.proto



<a name="rpc.EventsResendRequest"></a>

### EventsResendRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| event_id | [string](#string) |  | The ID of the event to resend. |
| account | [string](#string) |  | Resend the event to the given Stripe account. This is useful when testing a Connect platform. |
| data | [string](#string) | repeated | Additional data to send with an API request. Supports setting nested values (e.g nested[param]=value). |
| expand | [string](#string) | repeated | Response attributes to expand inline (target nested values with nested[param]=value). |
| idempotency | [string](#string) |  | Set an idempotency key for the request, preventing the same request from replaying within 24 hours. |
| live | [bool](#bool) |  | Make a live request (by default, runs in test mode). |
| stripe_account | [string](#string) |  | Specify the Stripe account to use for this request. |
| version | [string](#string) |  | Specify the Stripe API version to use for this request. |
| webhook_endpoint | [string](#string) |  | Resend the event to the given webhook endpoint ID. |






<a name="rpc.EventsResendResponse"></a>

### EventsResendResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stripe_event | [StripeEvent](#rpc.StripeEvent) |  |  |





 

 

 

 



<a name="fixtures.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## fixtures.proto



<a name="rpc.FixtureRequest"></a>

### FixtureRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| event | [string](#string) |  | An event to get the default fixture for |






<a name="rpc.FixtureResponse"></a>

### FixtureResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fixture | [string](#string) |  | default fixture for event |





 

 

 

 



<a name="listen.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## listen.proto



<a name="rpc.ListenRequest"></a>

### ListenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| connect_headers | [string](#string) | repeated | A list of custom headers to forward for Connect |
| events | [string](#string) | repeated | A list of specific events to listen for. For a list of all possible events, see: https://stripe.com/docs/api/events/types (default [*]) |
| forward_connect_to | [string](#string) |  | The URL to forward Connect webhook events to (default: same as normal events) |
| forward_to | [string](#string) |  | The URL to forward webhook events to |
| headers | [string](#string) | repeated | A list of custom headers to forward |
| latest | [bool](#bool) |  | Receive events formatted with the latest API version (default: your account&#39;s default API version) |
| live | [bool](#bool) |  | Receive live events (default: test) |
| skip_verify | [bool](#bool) |  | Skip certificate verification when forwarding to HTTPS endpoints |
| use_configured_webhooks | [bool](#bool) |  | Load webhook endpoint configuration from the webhooks API/dashboard |






<a name="rpc.ListenResponse"></a>

### ListenResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| state | [ListenResponse.State](#rpc.ListenResponse.State) |  | Check if the stream ready |
| stripe_event | [StripeEvent](#rpc.StripeEvent) |  | A Stripe event |
| endpoint_response | [ListenResponse.EndpointResponse](#rpc.ListenResponse.EndpointResponse) |  | A response from an endpoint |






<a name="rpc.ListenResponse.EndpointResponse"></a>

### ListenResponse.EndpointResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [ListenResponse.EndpointResponse.Data](#rpc.ListenResponse.EndpointResponse.Data) |  |  |
| error | [string](#string) |  |  |






<a name="rpc.ListenResponse.EndpointResponse.Data"></a>

### ListenResponse.EndpointResponse.Data



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [int64](#int64) |  | HTTP status code |
| http_method | [ListenResponse.EndpointResponse.Data.HttpMethod](#rpc.ListenResponse.EndpointResponse.Data.HttpMethod) |  | HTTP method |
| url | [string](#string) |  | URL of the webhook endpoint |
| event_id | [string](#string) |  | ID of the Stripe event that caused this response |





 


<a name="rpc.ListenResponse.EndpointResponse.Data.HttpMethod"></a>

### ListenResponse.EndpointResponse.Data.HttpMethod


| Name | Number | Description |
| ---- | ------ | ----------- |
| HTTP_METHOD_UNSPECIFIED | 0 |  |
| HTTP_METHOD_GET | 1 |  |
| HTTP_METHOD_POST | 2 |  |
| HTTP_METHOD_DELETE | 3 |  |



<a name="rpc.ListenResponse.State"></a>

### ListenResponse.State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| STATE_LOADING | 1 |  |
| STATE_RECONNECTING | 2 |  |
| STATE_READY | 3 |  |
| STATE_DONE | 4 |  |


 

 

 



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





 

 

 

 



<a name="logs_tail.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## logs_tail.proto



<a name="rpc.LogsTailRequest"></a>

### LogsTailRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter_accounts | [LogsTailRequest.Account](#rpc.LogsTailRequest.Account) | repeated | CONNECT ONLY* Filter request logs by source and destination account |
| filter_http_methods | [LogsTailRequest.HttpMethod](#rpc.LogsTailRequest.HttpMethod) | repeated | Filter request logs by http method |
| filter_ip_addresses | [string](#string) | repeated | Filter request logs by ip address |
| filter_request_paths | [string](#string) | repeated | Filter request logs by request path |
| filter_request_statuses | [LogsTailRequest.RequestStatus](#rpc.LogsTailRequest.RequestStatus) | repeated | Filter request logs by request status |
| filter_sources | [LogsTailRequest.Source](#rpc.LogsTailRequest.Source) | repeated | Filter request logs by source |
| filter_status_codes | [string](#string) | repeated | Filter request logs by status code |
| filter_status_code_types | [LogsTailRequest.StatusCodeType](#rpc.LogsTailRequest.StatusCodeType) | repeated | Filter request logs by status code type |






<a name="rpc.LogsTailResponse"></a>

### LogsTailResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| state | [LogsTailResponse.State](#rpc.LogsTailResponse.State) |  | Check if the stream ready |
| log | [LogsTailResponse.Log](#rpc.LogsTailResponse.Log) |  | A Stripe API log |






<a name="rpc.LogsTailResponse.Log"></a>

### LogsTailResponse.Log



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| livemode | [bool](#bool) |  |  |
| method | [string](#string) |  |  |
| url | [string](#string) |  |  |
| status | [int64](#int64) |  |  |
| request_id | [string](#string) |  |  |
| created_at | [int64](#int64) |  |  |
| error | [LogsTailResponse.Log.Error](#rpc.LogsTailResponse.Log.Error) |  |  |






<a name="rpc.LogsTailResponse.Log.Error"></a>

### LogsTailResponse.Log.Error



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| charge | [string](#string) |  |  |
| code | [string](#string) |  |  |
| decline_code | [string](#string) |  |  |
| message | [string](#string) |  |  |
| param | [string](#string) |  |  |





 


<a name="rpc.LogsTailRequest.Account"></a>

### LogsTailRequest.Account


| Name | Number | Description |
| ---- | ------ | ----------- |
| ACCOUNT_UNSPECIFIED | 0 |  |
| ACCOUNT_CONNECT_IN | 1 |  |
| ACCOUNT_CONNECT_OUT | 2 |  |
| ACCOUNT_SELF | 3 |  |



<a name="rpc.LogsTailRequest.HttpMethod"></a>

### LogsTailRequest.HttpMethod


| Name | Number | Description |
| ---- | ------ | ----------- |
| HTTP_METHOD_UNSPECIFIED | 0 |  |
| HTTP_METHOD_GET | 1 |  |
| HTTP_METHOD_POST | 2 |  |
| HTTP_METHOD_DELETE | 3 |  |



<a name="rpc.LogsTailRequest.RequestStatus"></a>

### LogsTailRequest.RequestStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| REQUEST_STATUS_UNSPECIFIED | 0 |  |
| REQUEST_STATUS_SUCCEEDED | 1 |  |
| REQUEST_STATUS_FAILED | 2 |  |



<a name="rpc.LogsTailRequest.Source"></a>

### LogsTailRequest.Source


| Name | Number | Description |
| ---- | ------ | ----------- |
| SOURCE_UNSPECIFIED | 0 |  |
| SOURCE_API | 1 |  |
| SOURCE_DASHBOARD | 2 |  |



<a name="rpc.LogsTailRequest.StatusCodeType"></a>

### LogsTailRequest.StatusCodeType


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_CODE_TYPE_UNSPECIFIED | 0 |  |
| STATUS_CODE_TYPE_2XX | 1 |  |
| STATUS_CODE_TYPE_4XX | 2 |  |
| STATUS_CODE_TYPE_5XX | 3 |  |



<a name="rpc.LogsTailResponse.State"></a>

### LogsTailResponse.State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| STATE_LOADING | 1 |  |
| STATE_RECONNECTING | 2 |  |
| STATE_READY | 3 |  |
| STATE_DONE | 4 |  |


 

 

 



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





 

 

 

 



<a name="trigger.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## trigger.proto



<a name="rpc.TriggerRequest"></a>

### TriggerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| event | [string](#string) |  | An event to trigger. Use `TriggersList` to see the available events. |
| stripe_account | [string](#string) |  | Set a header identifying the connected account |
| skip | [string](#string) | repeated | Skip specific steps in the fixture |
| override | [string](#string) | repeated | Override parameters in the fixture |
| add | [string](#string) | repeated | Add parameters in the fixture |
| remove | [string](#string) | repeated | Remove parameters from the fixture |
| raw | [string](#string) |  | Raw fixture string |






<a name="rpc.TriggerResponse"></a>

### TriggerResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| requests | [string](#string) | repeated | List of requests made during this trigger |





 

 

 

 



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





 

 

 

 



<a name="webhook_endpoints_list.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## webhook_endpoints_list.proto



<a name="rpc.WebhookEndpointsListRequest"></a>

### WebhookEndpointsListRequest







<a name="rpc.WebhookEndpointsListResponse"></a>

### WebhookEndpointsListResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoints | [WebhookEndpointsListResponse.WebhookEndpointData](#rpc.WebhookEndpointsListResponse.WebhookEndpointData) | repeated | A list webhook endpoints |






<a name="rpc.WebhookEndpointsListResponse.WebhookEndpointData"></a>

### WebhookEndpointsListResponse.WebhookEndpointData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| application | [string](#string) |  | Webhook endpoint application |
| enabledEvents | [string](#string) | repeated | Enabled events of the webhook endpoint |
| url | [string](#string) |  | Webhook endpoint URL |
| status | [string](#string) |  | Webhook endpoint status |





 

 

 

 



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

