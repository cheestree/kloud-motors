# computacao-nuvem-2025

## Run User and Seller Services

- Populate `.env` (rename from `.env.example`)

Start the user and seller containers:

```bash
docker compose up -d --build user seller user-db seller-db
```

Run the setup script with Python:

```bash
./venv/bin/python code/setup/setup_auth_db.py
```