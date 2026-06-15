# Cloudy Day Test Results - Daniel Carvalho

Tester: Daniel Alexandre Marto Queijo de Carvalho, 66208

Target API: European Country Metrics Aggregator

Test date: June 3, 2026

## Scope

This document reports my individual Cloudy Day tests. As required, I tested one endpoint with a smoke test and another endpoint with a stress test:

| Test type | Endpoint | Goal |
| --- | --- | --- |
| Smoke test | `POST /auth/login` | Verify authentication behavior for valid, invalid, incomplete, and malformed login requests. |
| Stress test | `GET /locations/top` | Evaluate the ranking endpoint under increasing concurrent load and validate sorting, limits, and response structure. |

The smoke test was implemented with a Python script using `httpx`. The stress test was implemented with Locust. Both tests validated HTTP responses and response bodies against the expected API behavior.

## Smoke Test - `POST /auth/login`

### Objective

The objective of this smoke test was to check whether the API correctly authenticates an existing user and handles common invalid login cases.

Since login requires an existing user, the test first created a user with `POST /auth/register`, then used the registered credentials to test `POST /auth/login`.

The main valid login payload was:

```json
{
  "username": "avalidusername4",
  "email": "avalidemail4@email.com"
}
```

### Validation Rules

Each scenario was considered successful only if the API returned the expected status code and, when applicable, a valid JSON body with the expected fields.

The smoke test checked the following cases:

- Registering a new user should return `201 Created`.
- Registering the same user again should return `409 Conflict`.
- Logging in with valid credentials should return `200 OK`.
- The successful login response should include the expected user object.
- Logging in with unknown credentials should return `401 Unauthorized`.
- Logging in with missing fields should return `400 Bad Request`.
- Registering with only an email should return `400 Bad Request`.
- Sending a wrong content type should return a client error, such as `400 Bad Request` or `415 Unsupported Media Type`.
- Cleanup should remove the test user successfully.

### Results

| Check | Result |
| --- | --- |
| Register new user | `201 Created` |
| Duplicate registration | `409 Conflict` |
| Login with valid credentials | `200 OK` |
| Login response schema | Failed: missing expected `User` object |
| Login with unknown credentials | `401 Unauthorized` |
| Login with missing username | `400 Bad Request` |
| Login with missing email | `400 Bad Request` |
| Login with empty body | `400 Bad Request` |
| Email-only registration | `400 Bad Request` |
| Wrong content type | Failed: returned `500 Internal Server Error` |
| Cleanup request | Failed: returned `404 Not Found` |
| Total result | `13/16 checks passed` |

### Smoke Test Conclusion

Most authentication behavior worked as expected. Registration succeeded, duplicate users were rejected, valid login returned `200 OK`, unknown users returned `401 Unauthorized`, and missing fields returned `400 Bad Request`.

However, the smoke test found three issues:

- The successful login response did not include the expected `User` object.
- A request with `Content-Type: text/plain` returned `500 Internal Server Error` instead of a client error.
- The cleanup request returned `404 Not Found`.

The endpoint is partially healthy, but the response schema and malformed request handling should be corrected before considering it production-ready.

## Stress Test - `GET /locations/top`

### Objective

The objective of this stress test was to evaluate how the top-locations endpoint behaves under increasing concurrent load.

According to the API behavior tested, `GET /locations/top` receives the query parameters `metric`, `order`, and `limit`. The `metric` parameter defines which metric is used for the ranking, `order` defines whether the result should be ascending or descending, and `limit` controls the maximum number of returned locations.

The stress test used multiple valid scenarios:

```http
GET /locations/top?metric=hpi&order=desc&limit=5
GET /locations/top?metric=crime&order=asc&limit=10
GET /locations/top?metric=life_expectancy&order=desc&limit=15
GET /locations/top?metric=pli&order=asc&limit=5
```

### Validation Rules

Each response was considered successful only if:

- The HTTP status code was `200 OK`.
- The response body was valid JSON.
- The response body was a non-empty array.
- Every returned object contained the required fields:
  - `rank`
  - `name`
  - `value`
- The number of returned locations did not exceed the requested `limit`.
- The returned values were correctly sorted according to the selected `order`.
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

This profile was useful to observe normal behavior, high-load behavior, a short concurrency spike, and recovery after the peak.

### Results

| Metric | Result |
| --- | --- |
| Total requests | `25502` |
| Failures | `0` |
| Error rate | `0%` |
| Median response time | `37ms` |
| Average response time | `66.6ms` |
| 90th percentile | `110ms` |
| 95th percentile | `190ms` |
| 99th percentile | `640ms` |
| Maximum response time | `2972ms` |
| Average throughput | `51.62 requests/s` |

The endpoint behaved correctly during the whole run. Locust recorded no unexpected status codes, invalid JSON responses, invalid response structures, sorting errors, limit violations, or latency failures above the configured `3000ms` threshold.

The median response time stayed low at `37ms`, and the 95th percentile stayed at `190ms`. The maximum response time reached `2972ms`, which was close to the configured threshold but still below it.

### Stress Test Conclusion

The stress test did not reveal any functional issue in `GET /locations/top`. The endpoint respected the tested API contract for all metric/order/limit combinations and handled up to `200` concurrent users without failures.

Although the endpoint passed the test, the maximum latency was very close to the `3000ms` stress threshold. This endpoint should continue to be monitored under longer test runs or heavier traffic, especially during spike scenarios.

## API Feedback

The two tested endpoints showed different maturity levels:

- `POST /auth/login` handles most normal and negative authentication cases correctly, but has schema consistency and malformed request handling issues.
- `GET /locations/top` is stable under the tested stress profile and follows the expected response structure, ordering, and limit rules.

Recommended improvements:

- Make the successful login response match the documented or expected schema by including the expected `User` object, or update the contract if the implementation is intentionally different.
- Return a client error such as `400 Bad Request` or `415 Unsupported Media Type` for unsupported content types instead of `500 Internal Server Error`.
- Review the cleanup/delete behavior used by the authentication test, since the cleanup request returned `404 Not Found`.
- Keep monitoring `GET /locations/top` under heavier or longer load tests because the maximum response time was close to the `3000ms` threshold.
