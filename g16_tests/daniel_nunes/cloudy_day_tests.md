# Cloudy Day Test Results

Daniel Ribeiro Nunes, fc66462

## Scope

This document reports the tests during Cloudy Day. The following endpoints were tested:

| Test type | Endpoint | Goal |
| --- | --- | --- |
| Smoke test | `GET /locations/{id}` | Verify successful location lookup and error handling for invalid or missing resources. |
| Stress test | `GET /locations/code/{code}` | Evaluate country-code lookup under increasing concurrent load and identify performance degradation. |

The smoke test was performed with Postman and validated both successful and invalid requests. The stress test was implemented with Locust and validated both HTTP responses and response bodies against the expected API contract.

## Smoke Test - `GET /locations/{id}`

### Objective

The objective of this smoke test was to verify that the API correctly retrieves the details of a specific location by numeric identifier.

This test also checks how the endpoint handles missing resources, invalid identifiers, query parameters, path variants and unsupported methods.

### Validation Rules

Each response was considered successful if:

- Existing numeric IDs returned `200 OK`;
- Successful responses contained the expected fields:
  - `id`
  - `name`
  - `country_code`
  - `metrics`
- Missing numeric IDs returned `404 Not Found`;
- Invalid ID formats returned `400 Bad Request`;
- The optional `year` query parameter filtered the `metrics` array when data existed;
- Invalid `year` values returned `400 Bad Request`;
- Unsupported methods returned the expected HTTP status.

### Tested Scenarios

| Scenario | Requests | Expected result |
| --- | --- | --- |
| Valid locations | `GET /locations/1`, `GET /locations/24`, `GET /locations/30` | `200 OK` with the expected location fields. |
| Missing locations | `GET /locations/0`, `GET /locations/-1`, `GET /locations/999999` | `404 Not Found` with `Location not found`. |
| Invalid IDs | `GET /locations/1.5`, `GET /locations/abc`, `GET /locations/1abc`, `GET /locations/%20`, `GET /locations/1%20OR%201=1` | `400 Bad Request`. |
| Valid year filter | `GET /locations/1?year=2024`, `GET /locations/1?year=2023`, `GET /locations/1?year=2015`, `GET /locations/1?year=2014` | `200 OK`, with metrics filtered to the requested year when data existed. |
| Years without data | `GET /locations/1?year=2030`, `GET /locations/1?year=1900`, `GET /locations/1?year=0`, `GET /locations/1?year=-1` | `200 OK` with an empty `metrics` array. |
| Invalid year filter | `GET /locations/1?year=abcd`, `GET /locations/1?year=2023.5` | `400 Bad Request`. |
| Extra query parameters | `GET /locations/1?foo=bar`, `GET /locations/1?year=2023&foo=bar` | `200 OK`; unknown parameters did not break the endpoint. |
| Path variants | `GET /locations/1/`, `GET /locations//1` | `404 Not Found`. |
| Unsupported methods | `HEAD /locations/1`, `DELETE /locations/1` | `HEAD` returned `200 OK`; `DELETE` returned `405 Method Not Allowed`. |

### Conclusion

The endpoint behaved correctly for the tested smoke scenarios. Existing locations returned the documented structure, missing resources and invalid IDs produced the expected error responses and no internal server errors were observed.

## Stress Test - `GET /locations/code/{code}`

### Objective

The objective of this stress test was to evaluate how the country-code lookup endpoint behaves under increasing concurrent load.

This endpoint fetches location data by country code and supports filtering results by year as well.

### Tested Request Pattern

Each virtual user sent requests to:

```http
GET /locations/code/{code}?year={year}
```

The test randomly selected a country code and a year from the following values:

| Parameter | Values |
| --- | --- |
| Country codes | `AT`, `BE`, `DE`, `ES`, `FR`, `IE`, `IT`, `NL`, `PT`, `SE` |
| Years | `2020`, `2021`, `2022`, `2023`, `2024` |

### Validation Rules

Each response was considered successful only if:

- The HTTP status code was `200 OK`.
- The response body was valid JSON.
- The response body contained the required fields:
  - `id`
  - `name`
  - `country_code`
  - `metrics`
- The returned `country_code` matched the requested code.
- The `metrics` field was an array.
- The `metrics` array was not empty for the tested years.
- Returned metric entries matched the requested year when year data was present.

### Load Profile

The stress test used a staged load profile. The number of virtual users increased by `100` every `30` seconds until reaching `1000` concurrent users. After the peak stage, the test ramped down to `0` users.

### Results

| Metric | Result |
| --- | --- |
| Total requests | `24762` |
| Failures | `1416` |
| Error rate | `5.72%` |
| Average throughput | `42.85 requests/s` |
| Average response time | `681.4ms` |
| Minimum response time | `38ms` |
| Median response time | `72ms` |
| 90th percentile | `1250ms` |
| 95th percentile | `3700ms` |
| 98th percentile | `7600ms` |
| 99th percentile | `9800ms` |
| Maximum response time | `21486.7ms` |

Overall, the endpoint remained functionally correct for most of the run, but the stress threshold was exceeded during the peak load stages. The recorded failures were caused by requests taking longer than the accepted latency limit.

The response time distribution shows that the service became saturated under heavier load. The median response time remained low at `72ms`, which means most requests were still fast. However, the tail latency increased significantly, with the 95th percentile reaching `3700ms` and the maximum response time reaching approximately `21.5s`.

### Stress Test Conclusion

The main limitation found was performance degradation under high concurrency. The failure ratio of `5.72%` and the high p95 and p99 response times show that the endpoint is stable in terms of correctness, but not yet comfortable at the level of load enforced by this test.

## API Feedback

The API is mostly consistent for the two endpoints I tested:

- `GET /locations/{id}` correctly handles successful lookups, missing resources, invalid identifiers, optional year filtering and unsupported methods.
- `GET /locations/code/{code}` follows the expected response structure for valid requests and remains functionally correct under load.

The most important improvement is performance optimization for `GET /locations/code/{code}` under high concurrency.

Recommended improvements:

- Optimize database access for country-code and year-filtered lookups;
- Add caching for repeated country-code queries if the underlying data does not change frequently.
