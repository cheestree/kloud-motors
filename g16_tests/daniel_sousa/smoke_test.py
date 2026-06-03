"""
SMOKE TEST — Migration Assistant API
Endpoint: GET /locations

Instalar: pip install locust

Correr (headless):
    locust -f smoke_test.py --headless \
           --users 1 --spawn-rate 1 --run-time 30s \
           --host https://<URL_DO_MOODLE>/api/v1 \
           --csv=logs/smoke --html=logs/smoke_report.html

Correr (com UI web em localhost:8089):
    locust -f smoke_test.py --host https://<URL_DO_MOODLE>/api/v1

Os resultados ficam em:
    logs/smoke_test.log        → log de checks (sucesso/falha por pedido)
    logs/smoke_stats.csv       → estatísticas agregadas (p50, p95, p99, RPS...)
    logs/smoke_stats_history.csv → evolução das stats ao longo do tempo
    logs/smoke_failures.csv    → pedidos que falharam
    logs/smoke_report.html     → relatório visual completo
"""

import logging
import os
from datetime import datetime
from locust import HttpUser, task, between, events

# ─── Logger para ficheiro ─────────────────────────────────────────────────────

os.makedirs("logs", exist_ok=True)

log_path = f"logs/smoke_test.log"

file_handler = logging.FileHandler(log_path, mode="w", encoding="utf-8")
file_handler.setLevel(logging.INFO)
file_handler.setFormatter(logging.Formatter("%(asctime)s [%(levelname)s] %(message)s"))

logger = logging.getLogger("smoke")
logger.setLevel(logging.INFO)
logger.addHandler(file_handler)
logger.addHandler(logging.StreamHandler())  # também imprime no terminal


# ─── Eventos globais ──────────────────────────────────────────────────────────

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    logger.info("=" * 60)
    logger.info("SMOKE TEST INICIADO")
    logger.info(f"Host: {environment.host}")
    logger.info(f"Data: {datetime.now().isoformat()}")
    logger.info("=" * 60)


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    stats = environment.stats.total
    logger.info("=" * 60)
    logger.info("SMOKE TEST CONCLUÍDO")
    logger.info(f"Total de pedidos : {stats.num_requests}")
    logger.info(f"Falhas           : {stats.num_failures}")
    logger.info(f"Taxa de erro     : {stats.fail_ratio * 100:.2f}%")
    logger.info(f"Latência média   : {stats.avg_response_time:.1f}ms")
    logger.info(f"Latência p95     : {stats.get_response_time_percentile(0.95):.1f}ms")
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


# ─── Classe de utilizador ─────────────────────────────────────────────────────

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
                resp.failure(f"Status esperado 200, obtido {resp.status_code}")
                return

            try:
                data = resp.json()
            except Exception:
                resp.failure("Body não é JSON válido")
                return

            if not isinstance(data, list) or len(data) == 0:
                resp.failure("Resposta devia ser um array não vazio")
                return

            required = {"id", "name", "country_code"}
            for item in data:
                missing = required - item.keys()
                if missing:
                    resp.failure(f"Item sem campos obrigatórios: {missing}")
                    return

            duration_ms = resp.elapsed.total_seconds() * 1000
            if duration_ms > 800:
                resp.failure(f"Latência {duration_ms:.0f}ms excede limite smoke (800ms)")
                return

            resp.success()
