#!/usr/bin/env python3
"""
Smoke test: POST /auth/register + POST /auth/login

Flow:
    1. Register a fresh test user (201)
    2. Verify 409 on duplicate registration
    3. Login with the registered user (200) — validate response shape
    4. Login with unknown credentials (401)
    5. Login/register with missing fields (400)
    6. Schema probe — email-only login (spec quirk: no password field)
    7. Wrong Content-Type (400/415)
    8. Cleanup — DELETE /delete/{userId}

Usage:
    python smoke_login.py <username> <email>

Example:
    python smoke_login.py testuser testuser@example.com

Requires:
    pip install httpx
"""

import sys
import asyncio
import httpx

BASE_URL = "http://8.233.225.83/api/v1"

# ── Result tracking ───────────────────────────────────────────────────────────

results: list[dict] = []


def passed(label: str, note: str | None = None) -> None:
    results.append({"label": label, "passed": True})
    suffix = f" — {note}" if note else ""
    print(f"  ✓ {label}{suffix}")


def failed(label: str, note: str | None = None) -> None:
    results.append({"label": label, "passed": False})
    suffix = f" — {note}" if note else ""
    print(f"  ✗ {label}{suffix}", file=sys.stderr)


def info(msg: str) -> None:
    print(f"  → {msg}")


# ── HTTP helpers ──────────────────────────────────────────────────────────────

async def post_json(client: httpx.AsyncClient, path: str, body: dict) -> httpx.Response:
    return await client.post(f"{BASE_URL}{path}", json=body)


async def post_raw(client: httpx.AsyncClient, path: str, content: str, content_type: str) -> httpx.Response:
    return await client.post(
        f"{BASE_URL}{path}",
        content=content,
        headers={"Content-Type": content_type},
    )


# ── Case 0: Register ──────────────────────────────────────────────────────────

async def test_register(client: httpx.AsyncClient, username: str, email: str) -> int | None:
    """Register a fresh user. Returns UserID on success, None on failure."""
    print("\n[0] Register new user — expect 201")
    res = await post_json(client, "/auth/register", {"username": username, "email": email})

    try:
        info(f"response body: {res.json()}")
    except Exception:
        info(f"response body: {res.text!r}")

    if res.status_code != 201:
        failed("status 201", f"got {res.status_code}")
        return None
    passed("status 201")

    try:
        body = res.json()
    except Exception:
        failed("response is valid JSON")
        return None
    passed("response is valid JSON")

    user_id   = body.get("id")
    user_name = body.get("userNName")
    reg_email = body.get("email")

    if not isinstance(user_id, int):
        failed("ID is an int", f"got {type(user_id).__name__}")
        return None
    passed("ID is an int")

    if not isinstance(user_name, str):
        failed("UserName is a string", f"got {type(user_name).__name__}")
    else:
        passed("UserName is a string")

    if not isinstance(reg_email, str):
        failed("Email is a string", f"got {type(reg_email).__name__}")
    else:
        passed("Email is a string")

    info(f"registered UserID={user_id}")
    return user_id


# ── Case 1: Duplicate registration ───────────────────────────────────────────

async def test_duplicate_register(client: httpx.AsyncClient, username: str, email: str) -> None:
    print("\n[1] Duplicate registration — expect 409")
    res = await post_json(client, "/auth/register", {"username": username, "email": email})
    if res.status_code == 409:
        passed("status 409")
    else:
        failed("status 409", f"got {res.status_code}")


# ── Case 2: Login happy path ──────────────────────────────────────────────────

async def test_login_happy_path(client: httpx.AsyncClient, username: str, email: str) -> None:
    print("\n[2] Login happy path — expect 200")

    # NOTE: LoginRequest has no password field — only username + email.
    # This is almost certainly a spec issue (copy-paste from RegisterRequest),
    # but we test what the spec says.
    info("Note: spec defines no password field on LoginRequest")

    res = await post_json(client, "/auth/login", {"username": username, "email": email})

    if res.status_code != 200:
        failed("status 200", f"got {res.status_code}")
        return
    passed("status 200")

    try:
        body = res.json()
    except Exception:
        failed("response is valid JSON")
        return
    passed("response is valid JSON")

    user = body.get("User")
    if not isinstance(user, dict):
        failed("body.User is an object", f"got {user!r}")
        return
    passed("body.User is an object")

    for field_name, expected_type in [("UserID", int), ("UserName", str), ("Email", str)]:
        val = user.get(field_name)
        if not isinstance(val, expected_type):
            failed(f"User.{field_name} is {expected_type.__name__}", f"got {type(val).__name__}")
        else:
            passed(f"User.{field_name} is {expected_type.__name__}")


# ── Case 3: Login — unknown user ──────────────────────────────────────────────

async def test_login_unknown(client: httpx.AsyncClient) -> None:
    print("\n[3] Login with unknown credentials — expect 401")
    res = await post_json(client, "/auth/login", {
        "username": "definitelynotauser_xqz",
        "email":    "nobody@nowhere.invalid",
    })
    if res.status_code == 401:
        passed("status 401")
    else:
        failed("status 401", f"got {res.status_code}")


# ── Case 4: Missing fields ────────────────────────────────────────────────────

async def test_missing_fields(client: httpx.AsyncClient, username: str, email: str) -> None:
    print("\n[4] Missing fields — expect 400")

    cases = [
        ({"username": username}, "username-only → 400"),
        ({"email":    email},    "email-only → 400"),
        ({},                     "empty body → 400"),
    ]
    for body, label in cases:
        res = await post_json(client, "/auth/login", body)
        if res.status_code == 400:
            passed(label)
        else:
            failed(label, f"got {res.status_code}")


# ── Case 5: Schema probe ──────────────────────────────────────────────────────

async def test_schema_probe(client: httpx.AsyncClient, email: str) -> None:
    print("\n[5] Schema probe — register with email-only (should 400)")
    res = await post_json(client, "/auth/register", {"email": email})
    if res.status_code == 400:
        passed("email-only register → 400")
    else:
        info(f"status {res.status_code} (200/201 would mean username isn't enforced)")


# ── Case 6: Wrong Content-Type ────────────────────────────────────────────────

async def test_wrong_content_type(client: httpx.AsyncClient, username: str, email: str) -> None:
    print("\n[6] Wrong Content-Type on login — expect 400 or 415")
    res = await post_raw(
        client, "/auth/login",
        content=f"username={username}&email={email}",
        content_type="text/plain",
    )
    if res.status_code in (400, 415):
        passed(f"status {res.status_code}")
    else:
        failed("400 or 415", f"got {res.status_code}")


# ── Case 7: Cleanup ───────────────────────────────────────────────────────────

async def test_cleanup(client: httpx.AsyncClient, user_id: int) -> None:
    print(f"\n[7] Cleanup — DELETE /delete/{user_id}")
    res = await client.delete(f"{BASE_URL}/delete/{user_id}")
    if res.status_code == 200:
        passed(f"user {user_id} deleted")
    else:
        failed(f"DELETE /delete/{user_id}", f"got {res.status_code} — manual cleanup may be needed")


# ── Runner ────────────────────────────────────────────────────────────────────

async def main(username: str, email: str) -> None:
    print(f"\nSmoke test: /auth/register + /auth/login")
    print(f'  username="{username}"  email="{email}"')
    print("─" * 50)

    user_id: int | None = None

    async with httpx.AsyncClient(timeout=10.0) as client:
        user_id = await test_register(client, username, email)

        if user_id is None:
            print("\n  Registration failed — skipping login tests", file=sys.stderr)
        else:
            await test_duplicate_register(client, username, email)
            await test_login_happy_path(client, username, email)
            await test_login_unknown(client)
            await test_missing_fields(client, username, email)
            await test_schema_probe(client, email)
            await test_wrong_content_type(client, username, email)
            await test_cleanup(client, user_id)

    total  = len(results)
    n_pass = sum(1 for r in results if r["passed"])
    print("\n" + "─" * 50)
    print(f"Result: {n_pass}/{total} checks passed")
    if n_pass < total:
        sys.exit(1)


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: smoke_login.py <username> <email>", file=sys.stderr)
        sys.exit(1)
    asyncio.run(main(sys.argv[1], sys.argv[2]))