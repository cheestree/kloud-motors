# Smoke
locust -f smoke_test.py --headless \
       --users 1 --spawn-rate 1 --run-time 30s \
       --host http://8.233.225.83/api/v1 \
       --csv=logs/smoke --html=logs/smoke_report.html

# # Stress
locust -f stress_test.py --headless \
       --host http://8.233.225.83/api/v1 \
       --csv=logs/stress --html=logs/stress_report.html