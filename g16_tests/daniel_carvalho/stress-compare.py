#!/usr/bin/env python3
"""
Stress test: GET /locations/top

Contract source:
    /home/cheese/Downloads/combinedAPI(1).yaml

Install:
    python3 -m pip install locust

Run headless:
    locust -f code/stress-smoke/stress-compare.py --headless \
        --host https://<MOODLE_URL>/api/v1 \
        --csv=logs/stress_top \
        --html=logs/stress_top_report.html

Run with the Locust web UI:
    locust -f code/stress-smoke/stress-compare.py \
        --host https://<MOODLE_URL>/api/v1

Optional environment variables:
    TOP_METRIC=hpi
    TOP_ORDER=desc
    TOP_LIMIT=5
    TOP_SCENARIOS=hpi:desc:5,crime:asc:10,life_expectancy:desc:15,pli:asc:5
    MAX_LATENCY_MS=3000

Do not pass --users or --spawn-rate for the full stress run; StressShape
controls them. For a short smoke run, --users/--spawn-rate/--run-time are
also supported.
"""

import logging
import os
from dataclasses import dataclass
from datetime import datetime

try:
    from locust import HttpUser, LoadTestShape, between, events, task
except ModuleNotFoundError as exc:
    if exc.name != "locust":
        raise
    raise SystemExit(
        "Locust is not installed. Run `python3 -m pip install locust`, then start "
        "this test with `locust -f code/stress-smoke/stress-compare.py "
        "--host https://<MOODLE_URL>/api/v1`."
    ) from exc


DEFAULT_HOST = os.getenv("STRESS_HOST", "https://<MOODLE_URL>/api/v1")
TOP_METRIC = os.getenv("TOP_METRIC", "hpi")
TOP_ORDER = os.getenv("TOP_ORDER", "desc")
TOP_LIMIT = os.getenv("TOP_LIMIT", "5")
TOP_SCENARIOS = os.getenv(
    "TOP_SCENARIOS",
    "hpi:desc:5,crime:asc:10,life_expectancy:desc:15,pli:asc:5",
)
MAX_LATENCY_MS = int(os.getenv("MAX_LATENCY_MS", "3000"))

ALLOWED_METRICS = {
    "pli",
    "hpi",
    "crime",
    "air_quality",
    "transport",
    "health",
    "unemployment",
    "working_hours",
    "life_expectancy",
}

ALLOWED_ORDERS = {"asc", "desc"}


@dataclass(frozen=True)
class TopScenario:
    label: str
    metric: str
    order: str
    limit: int

os.makedirs("logs", exist_ok=True)
log_path = "logs/stress_top.log"

logger = logging.getLogger("stress.top")
logger.setLevel(logging.INFO)
logger.propagate = False

if not logger.handlers:
    file_handler = logging.FileHandler(log_path, mode="w", encoding="utf-8")
    file_handler.setLevel(logging.INFO)
    file_handler.setFormatter(logging.Formatter("%(asctime)s [%(levelname)s] %(message)s"))

    stream_handler = logging.StreamHandler()
    stream_handler.setLevel(logging.INFO)
    stream_handler.setFormatter(logging.Formatter("%(message)s"))

    logger.addHandler(file_handler)
    logger.addHandler(stream_handler)


def normalize_limit(value: str) -> int:
    try:
        limit = int(value)
    except ValueError:
        return 5
    return min(max(limit, 1), 15)


def normalize_order(value: str) -> str:
    order = value.lower().strip()
    if order not in ALLOWED_ORDERS:
        return "asc"
    return order


def normalize_metric(value: str) -> str:
    metric = value.lower().strip()
    if metric not in ALLOWED_METRICS:
        return "hpi"
    return metric


def default_scenario() -> TopScenario:
    metric = normalize_metric(TOP_METRIC)
    order = normalize_order(TOP_ORDER)
    limit = normalize_limit(TOP_LIMIT)
    return TopScenario(
        label=f"{metric} {order} limit {limit}",
        metric=metric,
        order=order,
        limit=limit,
    )


def parse_scenarios(value: str) -> list[TopScenario]:
    scenarios: list[TopScenario] = []
    for raw_scenario in [part.strip() for part in value.split(",") if part.strip()]:
        parts = [part.strip() for part in raw_scenario.split(":")]
        if len(parts) != 3:
            continue

        metric = normalize_metric(parts[0])
        order = normalize_order(parts[1])
        limit = normalize_limit(parts[2])
        scenarios.append(
            TopScenario(
                label=f"{metric} {order} limit {limit}",
                metric=metric,
                order=order,
                limit=limit,
            )
        )

    return scenarios or [default_scenario()]


SCENARIOS = parse_scenarios(TOP_SCENARIOS)


def top_params(scenario: TopScenario) -> dict[str, str]:
    return {
        "metric": scenario.metric,
        "order": scenario.order,
        "limit": str(scenario.limit),
    }


def validate_top_response(data: object, scenario: TopScenario) -> str | None:
    if not isinstance(data, list):
        return "Response should be a JSON array"

    if not data:
        return "Response should contain at least 1 location"

    if len(data) > scenario.limit:
        return f"Response should not contain more than {scenario.limit} locations"

    values: list[float] = []
    required_fields = {"rank", "name", "value"}

    for item in data:
        if not isinstance(item, dict):
            return "Each top location should be a JSON object"

        missing_fields = required_fields - item.keys()
        if missing_fields:
            return f"Top location missing required fields: {sorted(missing_fields)}"

        if not isinstance(item.get("rank"), int) or item["rank"] < 1:
            return "Top location rank should be a positive integer"
        if not isinstance(item.get("name"), str) or not item["name"]:
            return "Top location name should be a non-empty string"
        if not isinstance(item.get("value"), (int, float)):
            return "Top location value should be numeric"

        values.append(float(item["value"]))

    if scenario.order == "asc" and values != sorted(values):
        return "Top location values should be sorted ascending"
    if scenario.order == "desc" and values != sorted(values, reverse=True):
        return "Top location values should be sorted descending"

    return None


@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    logger.info("=" * 60)
    logger.info("STRESS TEST STARTED")
    logger.info("Endpoint: GET /locations/top")
    logger.info("Host: %s", environment.host)
    logger.info("Date: %s", datetime.now().isoformat())
    logger.info("Scenarios: %s", [scenario.label for scenario in SCENARIOS])
    logger.info("Max latency: %sms", MAX_LATENCY_MS)
    logger.info("=" * 60)


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    stats = environment.stats.total
    logger.info("=" * 60)
    logger.info("STRESS TEST FINISHED - FINAL SUMMARY")
    logger.info("Total requests : %s", stats.num_requests)
    logger.info("Failures       : %s", stats.num_failures)
    logger.info("Error rate     : %.2f%%", stats.fail_ratio * 100)
    logger.info("Average latency: %.1fms", stats.avg_response_time)
    logger.info("Latency p50    : %.1fms", stats.get_response_time_percentile(0.50))
    logger.info("Latency p95    : %.1fms", stats.get_response_time_percentile(0.95))
    logger.info("Latency p99    : %.1fms", stats.get_response_time_percentile(0.99))
    logger.info("Max latency    : %.1fms", stats.max_response_time)
    logger.info("Average RPS    : %.2f", stats.total_rps)
    logger.info("Log saved at   : %s", log_path)
    logger.info("=" * 60)


@events.request.add_listener
def on_request(
    request_type,
    name,
    response_time,
    response_length,
    exception,
    context,
    **kwargs,
):
    if exception:
        logger.warning("FAIL | %s %s | %.0fms | %s", request_type, name, response_time, exception)
        return

    logger.info("OK   | %s %s | %.0fms | %sB", request_type, name, response_time, response_length)


@events.spawning_complete.add_listener
def on_spawning_complete(user_count, **kwargs):
    logger.info("[phase] Spawning complete - %s active VUs", user_count)


class StressTopUser(HttpUser):
    wait_time = between(1, 2)
    host = DEFAULT_HOST

    def on_start(self):
        self.scenario_index = 0

    def next_scenario(self) -> TopScenario:
        scenario = SCENARIOS[self.scenario_index % len(SCENARIOS)]
        self.scenario_index += 1
        return scenario

    @task
    def get_locations_top(self):
        scenario = self.next_scenario()
        with self.client.get(
            "/locations/top",
            params=top_params(scenario),
            name=f"GET /locations/top [{scenario.label}]",
            catch_response=True,
        ) as response:
            if response.status_code != 200:
                if response.status_code == 0:
                    response.failure("Request failed before an HTTP response; check host or network")
                    return
                response.failure(f"Expected status 200, got {response.status_code}")
                return

            try:
                data = response.json()
            except ValueError:
                response.failure("Response body is not valid JSON")
                return

            validation_error = validate_top_response(data, scenario)
            if validation_error:
                response.failure(validation_error)
                return

            duration_ms = response.elapsed.total_seconds() * 1000
            if duration_ms > MAX_LATENCY_MS:
                response.failure(
                    f"Latency {duration_ms:.0f}ms exceeds stress threshold ({MAX_LATENCY_MS}ms)"
                )
                return

            response.success()


class StressShape(LoadTestShape):
    """
    Each tuple is: (elapsed_seconds, target_users, spawn_rate).
    """

    use_common_options = True

    stages = [
        (30, 10, 1),
        (90, 10, 1),
        (120, 50, 5),
        (180, 50, 5),
        (210, 100, 10),
        (330, 100, 10),
        (340, 200, 50),
        (400, 200, 50),
        (430, 50, 10),
        (490, 50, 10),
        (520, 0, 10),
    ]

    def tick(self):
        run_time = self.get_run_time()
        options = self.runner.environment.parsed_options

        if options.run_time and run_time >= options.run_time:
            return None

        if options.num_users is not None and options.spawn_rate is not None:
            return options.num_users, options.spawn_rate

        for elapsed_seconds, users, spawn_rate in self.stages:
            if run_time < elapsed_seconds:
                return users, spawn_rate
        return None


if __name__ == "__main__":
    raise SystemExit(
        "Run this file with Locust, for example: "
        "`locust -f code/stress-smoke/stress-compare.py "
        "--host https://<MOODLE_URL>/api/v1`."
    )
