"""
STRESS TEST — Migration Assistant API
Endpoint: GET /locations/compare

Install: pip install locust

Run (headless):
    locust -f stress_test.py --headless \
           --host https://<MOODLE_URL>/api/v1 \
           --csv=logs/stress --html=logs/stress_report.html

Run (with web UI at localhost:8089):
    locust -f stress_test.py --host https://<MOODLE_URL>/api/v1

Results are saved in:
    logs/stress_test.log         → event log and phase summary
    logs/stress_stats.csv        → aggregated stats (p50, p95, p99, RPS...)
    logs/stress_stats_history.csv→ stats evolution over time
    logs/stress_failures.csv     → failed requests
    logs/stress_report.html      → full visual report

Note: do not pass --users or --spawn-rate; the LoadShape controls these.
"""

import logging
import os
from datetime import datetime
from locust import HttpUser, task, between, LoadTestShape, events

# ─── File logger ─────────────────────────────────────────────────────────────

os.makedirs("logs", exist_ok=True)

log_path = "logs/stress_test.log"

file_handler = logging.FileHandler(log_path, mode="w", encoding="utf-8")
file_handler.setLevel(logging.INFO)
file_handler.setFormatter(logging.Formatter("%(asctime)s [%(levelname)s] %(message)s"))

logger = logging.getLogger("stress")
logger.setLevel(logging.INFO)
logger.addHandler(file_handler)
logger.addHandler(logging.StreamHandler())  # also prints to the terminal


# ─── Global events ───────────────────────────────────────────────────────────

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    logger.info("=" * 60)
    logger.info("STRESS TEST STARTED")
    logger.info(f"Host: {environment.host}")
    logger.info(f"Date: {datetime.now().isoformat()}")
    logger.info("=" * 60)


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    stats = environment.stats.total
    logger.info("=" * 60)
    logger.info("STRESS TEST FINISHED — FINAL SUMMARY")
    logger.info(f"Total requests   : {stats.num_requests}")
    logger.info(f"Failures         : {stats.num_failures}")
    logger.info(f"Error rate       : {stats.fail_ratio * 100:.2f}%")
    logger.info(f"Average latency  : {stats.avg_response_time:.1f}ms")
    logger.info(f"Latency p50      : {stats.get_response_time_percentile(0.50):.1f}ms")
    logger.info(f"Latency p95      : {stats.get_response_time_percentile(0.95):.1f}ms")
    logger.info(f"Latency p99      : {stats.get_response_time_percentile(0.99):.1f}ms")
    logger.info(f"Max latency      : {stats.max_response_time:.1f}ms")
    logger.info(f"Average RPS      : {stats.total_rps:.2f}")
    logger.info(f"Log saved at     : {log_path}")
    logger.info("=" * 60)


@events.request.add_listener
def on_request(request_type, name, response_time, response_length,
               exception, context, **kwargs):
    if exception:
        logger.warning(f"FAIL | {request_type} {name} | {response_time:.0f}ms | {exception}")
    else:
        logger.info(f"OK   | {request_type} {name} | {response_time:.0f}ms | {response_length}B")


# ─── Spawning listener to log phase changes ─────────────────────────────────

@events.spawning_complete.add_listener
def on_spawning_complete(user_count, **kwargs):
    logger.info(f"[phase] Spawning complete — {user_count} VUs active")


# ─── User class ──────────────────────────────────────────────────────────────

class StressUser(HttpUser):
    wait_time = between(1, 2)
    host      = "https://<URL_DO_MOODLE>/api/v1"

    @task
    def get_locations_compare(self):
        with self.client.get(
            "/locations/compare",
            params={
                "ids": "1,2,3",
                "metrics": "hpi,crime,life_expectancy",
            },
            name="GET /locations/compare",
            catch_response=True,
        ) as resp:

            if resp.status_code != 200:
                resp.failure(f"Expected status 200, got {resp.status_code}")
                return

            try:
                data = resp.json()
            except Exception:
                resp.failure("Response body is not valid JSON")
                return

            if not isinstance(data, list) or len(data) < 2:
                resp.failure("Response should be an array with at least 2 locations")
                return

            required = {"id", "name", "country_code", "metrics"}
            expected_metrics = {"hpi", "crime", "life_expectancy"}
            for item in data:
                missing = required - item.keys()
                if missing:
                    resp.failure(f"Item missing required fields: {missing}")
                    return

                metrics = item.get("metrics")
                if not isinstance(metrics, list) or len(metrics) == 0:
                    resp.failure("Field 'metrics' should be a non-empty array")
                    return

                metric_names = {
                    metric.get("name") or metric.get("metric")
                    for metric in metrics
                    if isinstance(metric, dict)
                }
                if not expected_metrics.intersection(metric_names):
                    resp.failure("Metrics do not include any of the requested metrics")
                    return

            duration_ms = resp.elapsed.total_seconds() * 1000
            if duration_ms > 3000:
                resp.failure(f"Latency {duration_ms:.0f}ms exceeds stress limit (3000ms)")
                return

            resp.success()


# ─── Forma de carga (LoadShape) ───────────────────────────────────────────────

class StressShape(LoadTestShape):
    """
    Each row: (accumulated_seconds, target_VUs, spawn_rate).
    """
    stages = [
        (  30,  10,   1),   # slow ramp-up
        (  90,  10,   1),   # plateau — normal load
        ( 120,  50,   5),   # increase to high load
        ( 180,  50,   5),   # plateau — high load
        ( 210, 100,  10),   # increase to stress
        ( 330, 100,  10),   # plateau — real stress (2 min)
        ( 340, 200,  50),   # sudden spike (10s)
        ( 400, 200,  50),   # plateau — sustain spike (1 min)
        ( 430,  50,  10),   # recovery
        ( 490,  50,  10),   # plateau — system recovered?
        ( 520,   0,  10),   # ramp-down
    ]

    def tick(self):
        run_time = self.get_run_time()
        for t, users, spawn_rate in self.stages:
            if run_time < t:
                return (users, spawn_rate)
        return None
