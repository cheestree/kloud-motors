import random
from locust import HttpUser, task, between, LoadTestShape

class ForumStressUser(HttpUser):
    wait_time = between(1, 2)

    @task(3)
    def get_all_posts(self):
        with self.client.get("/forum/posts", name="GET /forum/posts", catch_response=True) as resp:
            if resp.status_code in [200, 404]:
                if resp.elapsed.total_seconds() * 1000 > 2000:
                    resp.failure(f"Latência muito alta: {resp.elapsed.total_seconds()*1000:.0f}ms")
                else:
                    resp.success()
            else:
                resp.failure(f"Erro inesperado: Status {resp.status_code}")

    @task(1)
    def get_filtered_posts(self):
        user_id = random.randint(1, 5)
        with self.client.get(f"/forum/posts?userId={user_id}", name="GET /forum/posts?userId=X", catch_response=True) as resp:
            if resp.status_code in [200, 404]:
                resp.success()
            else:
                resp.failure(f"Erro inesperado: Status {resp.status_code}")

class StressShape(LoadTestShape):
    stages = [
        (30, 10, 1), # ramp-up
        (90, 10, 1), # plateau
        (120, 50, 5), # subida
        (180, 50, 5), # plateau
        (210, 100, 10), # subida para stress
        (330, 100, 10), # plateau stress
        (340, 200, 50), # spike (pico máximo)
        (400, 200, 50), # plateau do spike
        (430, 50, 10), # recovery
        (490, 50, 10), # plateau
        (520, 0, 10), # ramp-down
    ]

    def tick(self):
        run_time = self.get_run_time()
        for t, users, spawn_rate in self.stages:
            if run_time < t:
                return (users, spawn_rate)
        return None