"""
STRESS TEST — Migration Assistant API
Endpoint: GET /locations/compare

Instalar: pip install locust

Correr (headless):
    locust -f stress_test.py --headless \
           --host https://<URL_DO_MOODLE>/api/v1 \
           --csv=logs/stress --html=logs/stress_report.html

Correr (com UI web em localhost:8089):
    locust -f stress_test.py --host https://<URL_DO_MOODLE>/api/v1

Os resultados ficam em:
    logs/stress_test.log        → log de eventos e sumário de cada fase
    logs/stress_stats.csv       → estatísticas agregadas (p50, p95, p99, RPS...)
    logs/stress_stats_history.csv → evolução das stats ao longo do tempo
    logs/stress_failures.csv    → pedidos que falharam
    logs/stress_report.html     → relatório visual completo

Nota: não passar --users nem --spawn-rate; o LoadShape controla isso.
"""

import logging
import os
from datetime import datetime
from locust import HttpUser, task, between, LoadTestShape, events

# ─── Logger para ficheiro ─────────────────────────────────────────────────────

os.makedirs("logs", exist_ok=True)

log_path = "logs/stress_test.log"

file_handler = logging.FileHandler(log_path, mode="w", encoding="utf-8")
file_handler.setLevel(logging.INFO)
file_handler.setFormatter(logging.Formatter("%(asctime)s [%(levelname)s] %(message)s"))

logger = logging.getLogger("stress")
logger.setLevel(logging.INFO)
logger.addHandler(file_handler)
logger.addHandler(logging.StreamHandler())  # também imprime no terminal


# ─── Eventos globais ──────────────────────────────────────────────────────────

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    logger.info("=" * 60)
    logger.info("STRESS TEST INICIADO")
    logger.info(f"Host: {environment.host}")
    logger.info(f"Data: {datetime.now().isoformat()}")
    logger.info("=" * 60)


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    stats = environment.stats.total
    logger.info("=" * 60)
    logger.info("STRESS TEST CONCLUÍDO — SUMÁRIO FINAL")
    logger.info(f"Total de pedidos : {stats.num_requests}")
    logger.info(f"Falhas           : {stats.num_failures}")
    logger.info(f"Taxa de erro     : {stats.fail_ratio * 100:.2f}%")
    logger.info(f"Latência média   : {stats.avg_response_time:.1f}ms")
    logger.info(f"Latência p50     : {stats.get_response_time_percentile(0.50):.1f}ms")
    logger.info(f"Latência p95     : {stats.get_response_time_percentile(0.95):.1f}ms")
    logger.info(f"Latência p99     : {stats.get_response_time_percentile(0.99):.1f}ms")
    logger.info(f"Latência máx     : {stats.max_response_time:.1f}ms")
    logger.info(f"RPS médio        : {stats.total_rps:.2f}")
    logger.info(f"Log guardado em  : {log_path}")
    logger.info("=" * 60)


@events.request.add_listener
def on_request(request_type, name, response_time, response_length,
               exception, context, **kwargs):
    if exception:
        logger.warning(f"FAIL | {request_type} {name} | {response_time:.0f}ms | {exception}")
    else:
        logger.info(f"OK   | {request_type} {name} | {response_time:.0f}ms | {response_length}B")


# ─── Listener de spawning para logar mudanças de fase ────────────────────────

@events.spawning_complete.add_listener
def on_spawning_complete(user_count, **kwargs):
    logger.info(f"[fase] Spawning completo — {user_count} VUs ativos")


# ─── Classe de utilizador ─────────────────────────────────────────────────────

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
                resp.failure(f"Status esperado 200, obtido {resp.status_code}")
                return

            try:
                data = resp.json()
            except Exception:
                resp.failure("Body não é JSON válido")
                return

            if not isinstance(data, list) or len(data) < 2:
                resp.failure("Resposta devia ser um array com pelo menos 2 localizações")
                return

            required = {"id", "name", "country_code", "metrics"}
            expected_metrics = {"hpi", "crime", "life_expectancy"}
            for item in data:
                missing = required - item.keys()
                if missing:
                    resp.failure(f"Item sem campos obrigatórios: {missing}")
                    return

                metrics = item.get("metrics")
                if not isinstance(metrics, list) or len(metrics) == 0:
                    resp.failure("Campo metrics devia ser um array não vazio")
                    return

                metric_names = {
                    metric.get("name") or metric.get("metric")
                    for metric in metrics
                    if isinstance(metric, dict)
                }
                if not expected_metrics.intersection(metric_names):
                    resp.failure("Metrics não inclui nenhuma das métricas pedidas")
                    return

            duration_ms = resp.elapsed.total_seconds() * 1000
            if duration_ms > 3000:
                resp.failure(f"Latência {duration_ms:.0f}ms excede limite stress (3000ms)")
                return

            resp.success()


# ─── Forma de carga (LoadShape) ───────────────────────────────────────────────

class StressShape(LoadTestShape):
    """
    Cada linha: (segundos_acumulados, VUs_alvo, spawn_rate).
    """
    stages = [
        (  30,  10,   1),   # ramp-up lento
        (  90,  10,   1),   # plateau — carga normal
        ( 120,  50,   5),   # subida para carga alta
        ( 180,  50,   5),   # plateau — carga alta
        ( 210, 100,  10),   # subida para stress
        ( 330, 100,  10),   # plateau — stress real (2 min)
        ( 340, 200,  50),   # spike súbito (10s)
        ( 400, 200,  50),   # plateau — aguentar o spike (1 min)
        ( 430,  50,  10),   # recovery
        ( 490,  50,  10),   # plateau — sistema recuperou?
        ( 520,   0,  10),   # ramp-down
    ]

    def tick(self):
        run_time = self.get_run_time()
        for t, users, spawn_rate in self.stages:
            if run_time < t:
                return (users, spawn_rate)
        return None
