# Cloudy Day Test Results - Daniel Sousa

Tester: Daniel Martins Cabrita de Sousa, 66128

Target API: European Country Metrics Aggregator

Test date: June 3, 2026

## Scope

This document reports my individual Cloudy Day tests. As required, I tested one endpoint with a smoke test and another endpoint with a stress test:

| Test type | Endpoint | Goal |
| --- | --- | --- |
| Smoke test | `GET /locations` | Verify the main successful use case and response schema. |
| Stress test | `GET /locations/compare` | Evaluate the endpoint under increasing concurrent load and identify the practical request limit. |

The tests were implemented with Locust and validated both HTTP responses and response bodies against the API contract described in `combinedAPI.yaml`.

## Smoke Test - `GET /locations`

### Objective

The objective of this smoke test was to check if the API correctly returns the list of European locations available in the platform.

According to the API specification, `GET /locations` should return `200 OK` with a JSON array of location summaries.

### Validation Rules

Each response was considered successful only if:

- The HTTP status code was `200 OK`.
- The response body was valid JSON.
- The response body was a non-empty array.
- Every location object contained the required fields:
  - `id`
  - `name`
  - `country_code`
- The response time stayed below the smoke threshold of `800ms`.

### Results

| Metric | Result |
| --- | --- |
| Total requests | `20` |
| Failures | `0` |
| Error rate | `0%` |
| Median response time | `47ms` |
| Average response time | `62.7ms` |
| 95th percentile | `230ms` |
| Maximum response time | `231.2ms` |
| Average response size | `1406B` |

### Smoke Test Conclusion

The endpoint behaved as expected. All requests returned `200 OK`, the response body matched the documented structure, and no functional failures were detected.

The endpoint is healthy for the tested smoke scenario. Response times were comfortably below the configured `800ms` threshold.

## Stress Test - `GET /locations/compare`

### Objective

The objective of this stress test was to evaluate how the comparison endpoint behaves under increasing concurrent load.

According to the API specification, `GET /locations/compare` requires the query parameter `ids`, with at least two comma-separated location identifiers. The optional `metrics` parameter can restrict the comparison to specific metrics.

The tested request was:

```http
GET /locations/compare?ids=1,2,3&metrics=hpi,crime,life_expectancy
```

### Validation Rules

Each response was considered successful only if:

- The HTTP status code was `200 OK`.
- The response body was valid JSON.
- The response body was an array with at least two locations.
- Every location object contained the required fields:
  - `id`
  - `name`
  - `country_code`
  - `metrics`
- The returned metrics included at least one of the requested metrics:
  - `hpi`
  - `crime`
  - `life_expectancy`
- The response time stayed below the stress threshold of `3000ms`.

### Load Profile

The stress test used a staged load profile:

| Stage | Target load |
| --- | --- |
| Slow ramp-up | `10` virtual users |
| Normal load plateau | `10` virtual users |
| High load | `50` virtual users |
| Stress load | `100` virtual users |
| Spike | `200` virtual users |
| Recovery | `50` virtual users |
| Ramp-down | `0` virtual users |

This profile was useful to observe normal behavior, degradation under stress, a short spike, and recovery after peak load.

### Results

| Metric | Result |
| --- | --- |
| Total requests | `18858` |
| Failures | `1067` |
| Error rate | `5.66%` |
| Median response time | `57ms` |
| Average response time | `607.1ms` |
| 90th percentile | `1200ms` |
| 95th percentile | `3600ms` |
| 99th percentile | `9900ms` |
| Maximum response time | `22546.9ms` |
| Average throughput | `38.16 requests/s` |

The endpoint remained functionally correct during the test. The recorded failures were caused by responses exceeding the configured `3000ms` stress threshold, not by invalid status codes or invalid response bodies.

The median response time stayed low at `57ms`, which means most requests were fast. However, the upper percentiles show clear degradation under heavier load. The 95th percentile reached `3600ms`, already above the stress threshold, and the maximum response time reached approximately `22.5s`.

The failure log confirms this pattern with repeated latency failures such as responses taking more than `3000ms`.

### Stress Test Conclusion

The stress test did not reveal a specification mismatch for `GET /locations/compare`. The endpoint returned the expected data structure for valid comparison requests.

The main issue is performance under high concurrency. When the load reached the stress and spike phases, a relevant number of requests took longer than `3000ms`. This suggests that the comparison endpoint, or the data access layer behind it, should be optimized before the system is expected to support around `100` to `200` concurrent users.

## API Feedback

The API is mostly consistent for the two endpoints I tested:

- `GET /locations` is stable and follows the specification for the successful list scenario.
- `GET /locations/compare` follows the specification for valid requests and returns the expected structure.

The most important improvement is performance optimization for `GET /locations/compare`. The endpoint should be reviewed for expensive queries, missing indexes, inefficient joins, repeated calculations, or avoidable work done for every request.

Recommended improvements:

- Optimize database access for the comparison endpoint.
- Add caching for repeated comparisons or metric lookups, if the data does not change frequently.
- Review indexes for location IDs, country codes, years, and metric names.
- Add load monitoring for latency percentiles, not only average response time.


