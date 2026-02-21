import pytest
import requests
import time

def test_realtime_sse_scores(scorekeeper_client):
    # 1. Connect to the SSE endpoint /events/stream as an unauthenticated user or scorekeeper
    # 2. Scorekeeper submits a score update via API
    # 3. Verify the SSE stream receives the score update within a reasonable timeframe (e.g. 2s)
    pass
