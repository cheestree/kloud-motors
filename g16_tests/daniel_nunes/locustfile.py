import logging
import os
from datetime import datetime
from random import choice
from locust import FastHttpUser, LoadTestShape, between, events, task


os.makedirs("logs", exist_ok=True)

log_path = "logs/location_code_stress.log"

file_handler = logging.FileHandler(log_path, mode="w", encoding="utf-8")
file_handler.setLevel(logging.INFO)
file_handler.setFormatter(logging.Formatter("%(asctime)s [%(levelname)s] %(message)s"))

logger = logging.getLogger("location_code_stress")
logger.setLevel(logging.INFO)
logger.handlers.clear()
logger.propagate = False
logger.addHandler(file_handler)
logger.addHandler(logging.StreamHandler())


@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    logger.info("=" * 60)
    logger.info("STRESS TEST STARTED")
    logger.info(f"Endpoint: GET /locations/code/{{code}}")
    logger.info(f"Host: {environment.host}")
    logger.info(f"Date: {datetime.now().isoformat()}")
    logger.info("=" * 60)

@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    stats = environment.stats.total
    logger.info("=" * 60)
    logger.info("STRESS TEST FINISHED - FINAL SUMMARY")
    logger.info(f"Total requests : {stats.num_requests}")
    logger.info(f"Failures       : {stats.num_failures}")
    logger.info(f"Failure ratio  : {stats.fail_ratio * 100:.2f}%")
    logger.info(f"Average latency: {stats.avg_response_time:.1f}ms")
    logger.info(f"Latency p50    : {stats.get_response_time_percentile(0.50):.1f}ms")
    logger.info(f"Latency p95    : {stats.get_response_time_percentile(0.95):.1f}ms")
    logger.info(f"Latency p99    : {stats.get_response_time_percentile(0.99):.1f}ms")
    logger.info(f"Max latency    : {stats.max_response_time:.1f}ms")
    logger.info(f"Average RPS    : {stats.total_rps:.2f}")
    logger.info(f"Log saved in   : {log_path}")
    logger.info("=" * 60)

@events.request.add_listener
def on_request(request_type, name, response_time, response_length, exception, context, **kwargs):
    pass

@events.spawning_complete.add_listener
def on_spawning_complete(user_count, **kwargs):
    logger.info(f"[stage] Spawning complete - {user_count} active VUs")


class StressUser(FastHttpUser):
    wait_time = between(0.01, 0.05)
    host = "http://8.233.225.83/api/v1"

    country_codes = ("AT", "BE", "DE", "ES", "FR", "IE", "IT", "NL", "PT", "SE")
    years = (2020, 2021, 2022, 2023, 2024)

    @task
    def get_location_by_code(self):
        code = choice(self.country_codes)
        year = choice(self.years)

        with self.client.get(
            f"/locations/code/{code}",
            params={"year": year},
            name="GET /locations/code/{code}",
            catch_response=True,
        ) as response:
            if response.status_code != 200:
                response.failure(f"Expected 200, got {response.status_code}")
                return

            try:
                data = response.json()
            except ValueError:
                response.failure("Body is not valid JSON")
                return

            required = {"id", "name", "country_code", "metrics"}
            missing = required - data.keys()
            if missing:
                response.failure(f"Missing required fields: {missing}")
                return

            if data.get("country_code") != code:
                response.failure(f"Expected country_code {code}, got {data.get('country_code')}")
                return

            metrics = data.get("metrics")
            if not isinstance(metrics, list):
                response.failure("Field metrics should be an array")
                return

            if len(metrics) == 0:
                response.failure("Field metrics should not be empty for tested years")
                return

            metric_years = {
                metric.get("year")
                for metric in metrics
                if isinstance(metric, dict) and "year" in metric
            }
            if metric_years and metric_years != {year}:
                response.failure(f"Expected only metrics for {year}, got {sorted(metric_years)}")
                return

            response.success()


class StressShape(LoadTestShape):
    """
    Each row: (cumulative_seconds, target_users, spawn_rate).
    """
    stages = [
        (30, 100, 50),
        (60, 200, 50),
        (90, 300, 50),
        (120, 400, 50),
        (150, 500, 50),
        (180, 600, 50),
        (210, 700, 50),
        (240, 800, 50),
        (270, 900, 50),
        (300, 1000, 50),
        (330, 0, 100),
    ]

    def tick(self):
        run_time = self.get_run_time()
        for stage_time, users, spawn_rate in self.stages:
            if run_time < stage_time:
                return users, spawn_rate
        return None
