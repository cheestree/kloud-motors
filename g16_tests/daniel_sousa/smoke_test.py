"""
SMOKE TEST — Migration Assistant API
Endpoint: GET /locations

Install: pip install locust

Run (headless):
    locust -f smoke_test.py --headless \
           --users 1 --spawn-rate 1 --run-time 30s \
           --host https://<MOODLE_URL>/api/v1 \
           --csv=logs/smoke --html=logs/smoke_report.html

Run (with web UI at localhost:8089):
    locust -f smoke_test.py --host https://<MOODLE_URL>/api/v1

Results are saved in:
    logs/smoke_test.log         → check log (success/failure per request)
    logs/smoke_stats.csv        → aggregated stats (p50, p95, p99, RPS...)
    logs/smoke_stats_history.csv→ stats evolution over time
    logs/smoke_failures.csv     → failed requests
    logs/smoke_report.html      → full visual report
"""

import logging
import os
from datetime import datetime
from locust import HttpUser, task, between, events

# ─── File logger ─────────────────────────────────────────────────────────────

os.makedirs("logs", exist_ok=True)

log_path = f"logs/smoke_test.log"

file_handler = logging.FileHandler(log_path, mode="w", encoding="utf-8")
file_handler.setLevel(logging.INFO)
file_handler.setFormatter(logging.Formatter("%(asctime)s [%(levelname)s] %(message)s"))

logger = logging.getLogger("smoke")
logger.setLevel(logging.INFO)
logger.addHandler(file_handler)
logger.addHandler(logging.StreamHandler())  # também imprime no terminal


# ─── Global events ───────────────────────────────────────────────────────────

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    logger.info("=" * 60)
    logger.info("SMOKE TEST STARTED")
    logger.info(f"Host: {environment.host}")
    logger.info(f"Date: {datetime.now().isoformat()}")
    logger.info("=" * 60)


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    stats = environment.stats.total
    logger.info("=" * 60)
    logger.info("SMOKE TEST FINISHED")
    logger.info(f"Total requests   : {stats.num_requests}")
    logger.info(f"Failures         : {stats.num_failures}")
    logger.info(f"Error rate       : {stats.fail_ratio * 100:.2f}%")
    logger.info(f"Average latency  : {stats.avg_response_time:.1f}ms")
    logger.info(f"Latency p95      : {stats.get_response_time_percentile(0.95):.1f}ms")
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


# ─── User class ──────────────────────────────────────────────────────────────

class SmokeUser(HttpUser):
    wait_time = between(1, 2)
    host      = "https://<URL_DO_MOODLE>/api/v1"

    @task
    def get_locations(self):
        with self.client.get(
            "/locations",
            name="GET /locations",
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

            if not isinstance(data, list) or len(data) == 0:
                resp.failure("Response should be a non-empty array")
                return

            required = {"id", "name", "country_code"}
            for item in data:
                missing = required - item.keys()
                if missing:
                    resp.failure(f"Item missing required fields: {missing}")
                    return

            duration_ms = resp.elapsed.total_seconds() * 1000
            if duration_ms > 800:
                resp.failure(f"Latency {duration_ms:.0f}ms exceeds smoke limit (800ms)")
                return

            resp.success()
