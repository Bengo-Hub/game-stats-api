import os
import pytest
import psycopg2
import requests

# Base URLs
API_URL = os.getenv("TEST_API_URL", "http://localhost:8080/api/v1")
DB_HOST = os.getenv("TEST_DB_HOST", "localhost")
DB_PORT = os.getenv("TEST_DB_PORT", "5432")
DB_NAME = os.getenv("TEST_DB_NAME", "game_stats")
DB_USER = os.getenv("TEST_DB_USER", "postgres")
DB_PASS = os.getenv("TEST_DB_PASS", "postgres")

@pytest.fixture(scope="session")
def db_connection():
    """Provides a raw psycopg2 database connection for data seeding/verification."""
    conn = psycopg2.connect(
        host=DB_HOST,
        port=DB_PORT,
        dbname=DB_NAME,
        user=DB_USER,
        password=DB_PASS
    )
    conn.autocommit = True
    yield conn
    conn.close()

@pytest.fixture(scope="session")
def api_client():
    """Provides a requests session configured with the base API URL."""
    class ApiClient(requests.Session):
        def request(self, method, url, *args, **kwargs):
            full_url = f"{API_URL}{url}" if url.startswith("/") else url
            return super().request(method, full_url, *args, **kwargs)
    
    session = ApiClient()
    yield session

@pytest.fixture
def admin_client(api_client, db_connection):
    """Provides an API client authenticated as an admin user."""
    # Ensure admin user exists in DB via raw SQL if needed, or rely on fixtures.
    # We use the scorekeeper seeded earlier or the superadmin from fixtures.
    payload = {"username": "admin@codevertexitsolutions.com", "password": "password123"}
    resp = api_client.post("/login", json=payload)
    if resp.status_code == 200:
        token = resp.json().get("token")
        api_client.headers.update({"Authorization": f"Bearer {token}"})
    return api_client

@pytest.fixture
def scorekeeper_client(api_client):
    """Provides an API client authenticated as a scorekeeper."""
    payload = {"username": "scorekeeper", "password": "password123"}
    resp = api_client.post("/login", json=payload)
    if resp.status_code == 200:
        token = resp.json().get("token")
        api_client.headers.update({"Authorization": f"Bearer {token}"})
    return api_client
