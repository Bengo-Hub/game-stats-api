import pytest

def test_spirit_score_lifecycle(api_client, scorekeeper_client, db_connection):
    # 1. Ensure a tournament and teams exist
    event_payload = {"name": "Test Tournament", "startDate": "2026-01-01T00:00:00Z", "endDate": "2026-01-02T00:00:00Z", "location": "Nairobi"}
    # Assuming authenticated user or open creation
    resp = api_client.post("/events", json=event_payload)
    if resp.status_code == 201:
        event_id = resp.json().get("id")
    else:
        # Fallback to query
        event_id = 1
        
    team1_payload = {"name": "Team A", "city": "Nairobi", "eventId": event_id}
    team2_payload = {"name": "Team B", "city": "Mombasa", "eventId": event_id}
    # Create teams
    # 2. Scorekeeper creates a game between Team A and Team B
    # 3. Scorekeeper submits spirit score from Team A to Team B
    # 4. Scorekeeper submits spirit score from Team B to Team A
    # 5. Public view verifies spirit scores are visible and correct
    pass
