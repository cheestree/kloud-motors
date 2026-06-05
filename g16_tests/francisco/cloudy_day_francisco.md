# Cloudy Day Test Results - Francisco Encarnação

Tester: Francisco Lopes da Encarnação, 66131

Target API: European Country Metrics Aggregator

Test date: June 3, 2026



| Test type | Endpoint | Goal |
| --- | --- | --- |
| Smoke test | `POST /forum/posts` | Validate whether the API correctly processes valid posts and rejects invalid ones. |
| Stress test | `GET /forum/posts` | Evaluate the endpoint behavior under high volume, including global vs. parameterized requests. |

## Smoke Test - POST /forum/posts

### Objective
The goal of this endpoint is to create a new post in the forum. The test evaluated whether the API correctly processes valid posts and rejects invalid ones.

For this test, the tool used was Postman and according to the API specification, this endpoint has 2 possible scenarios:

### Validation Rules & Scenarios

The tests were executed using Postman, evaluating the following conditions:

- Scenario 1: Valid Post - Should return 201 Created.

- Scenario 2: Missing Fields - Should return 400 Bad Request.

- Security Check: Validate if post creation requires user authentication.

### Results
#### Specification Problem (AuthorID vs AuthorId)
When sending a valid JSON payload matching the specification (authorID), the API returned a 400 Bad Request saying the author ID was missing. Afterwards, the same request was sent but with the key authorId (lowercase 'd'), that was successful. This revealed a case-sensitivity mismatch between the specification and the implementation.

#### Missing Fields Validation
When sending payloads with missing fields, the API successfully identified the issue and returned a 400 Bad Request with an appropriate error message (e.g., "title: Title is required").

#### Security Validation
It was discovered that anyone can create a post without being authenticated. If a user guesses another user's ID, they can create posts on their behalf.

### Conclusion
Even though the `POST /forum/posts` endpoint correctly validates missing data and successfully creates posts when the correct case is used, it has a minor documentation mismatch and a major security flaw. A mechanism where the user needs a login token to create a post is required.

## Stress Test - GET /forum/posts

### Objective
The main goal of this test is to evaluate the behavior of the API when it is subjected to a high volume of requests. The test was conducted using the Locust tool, with a load profile that simulates a peak usage scenario.

According to the API specification, the request gets the forum posts and this can be done in two ways: without the userId parameter (global endpoint) or with the userId parameter (filtered endpoint). 

#### Load Profile
The stress test used a staged load profile. The test started with a ramp-up to 10 virtual users, increased to 50 users, then to 100 users, and finally peaked at 200 users before ramping down.

To properly diagnose the system's behavior, the stress testing was divided into two distinct execution phases.

### Phase 1: Test & Bug Discovery

In the first phase, the test generated traffic to both the global endpoint (`GET /api/v1/forum/posts`) and the filtered endpoint (`GET /api/v1/forum/posts?userId=X`).

The global endpoint showed a 100% failure rate and the filtered endpoint worked as expected.

In total, 17,931 requests were sent to the global endpoint and all failed. The API specification explicitly defines the userId query parameter as optional however, omitting it causes a crash. As the response times were impacted by the high failure rate, the 95th percentile showed latency spikes.

| Metric | Result |
| --- | --- |
| Total requests | `17931` |
| Failures | `17931` |
| Error rate | `100%` |
| Median response time | `45ms` |
| Average response time | `148.02ms` |
| 95th percentile | `620ms` |
| 99th percentile | `1500ms` |
| Maximum response time | `3878ms` |
| Average throughput | `22.7 requests/s` |

### Phase 2: Filtered Route Only

After understanding the issue, I modified the test to only send requests to the filtered endpoint (`GET /api/v1/forum/posts?userId=X`).

The system handled the initial ramp-up phases fine. However, during the peak load phase, the server reached its processing limit and began to drop requests.

As shown in the statistics below, the test generated over 18,700 requests, of which 1,337 failed, resulting in a 7% failure rate. This indicates that the application logic works correctly, but the infrastructure or database connection pool cannot sustain 200 concurrent users performing read operations.

The response times were also affected by the high load. The average response time was around 614ms, with the 99th percentile reaching 5400ms and a maximum peak of 17,740ms. These latency spikes correlate directly with the dropped requests visible on the failure rate graph.

| Metric | Result |
| --- | --- |
| Total requests | `18748` |
| Failures | `1377` |
| Error rate | `7.34%` |
| Median response time | `120ms` |
| Average response time | `613.99ms` |
| 95th percentile | `2400ms` |
| 99th percentile | `5400ms` |
| Maximum response time | `17740ms` |
| Average throughput | `30.6 requests/s` |

### Stress Test Conclusion

The stress testing revealed two issues in the target API:

1. The global endpoint (`GET /forum/posts`) is broken and returns a 500 Internal Server Error response.
2. The filtered endpoint (`GET /forum/posts?userId=X`) is implemented well but cannot handle high concurrency.

For that reason I suggest fixing the global endpoint and increasing the infrastructure resources to handle the high load.

## API Feedback
Based on the smoke and stress tests performed, the Forum endpoints require several critical fixes to be considered production-ready.

Recommended improvements:

- **Security**: Implement authentication and authorization checks on POST /forum/posts so users can only post on their own behalf using a secure token.

- **Documentation**: Correct the API specification regarding the authorId field casing.

- **Bug Fix**: Fix the backend logic for GET /forum/posts to ensure it gracefully handles requests when the optional userId parameter is omitted, instead of crashing (500 Error).

- **Performance**: Increase infrastructure resources or optimize the database connection pool so the filtered GET endpoint can sustain at least 200 concurrent read operations without dropping requests or causing severe latency spikes.