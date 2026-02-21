import pytest
import uuid
import datetime

def test_spirit_score_lifecycle(admin_client, scorekeeper_client, db_connection):
    event_payload = {
        "name": f"Spirit Test Tournament {uuid.uuid4().hex[:8]}",
        "slug": f"spirit-test-{uuid.uuid4().hex[:8]}",
        "year": 2026,
        "startDate": "2026-06-01T00:00:00Z",
        "endDate": "2026-06-02T00:00:00Z",
        "status": "draft",
        "categories": ["outdoor"],
        "teamsCount": 16,
        "gamesCount": 30
    }
    resp = admin_client.post("/admin/events", json=event_payload)
    assert resp.status_code == 201, f"Failed to create event: {resp.text}"
    event_id = resp.json().get("id")

    # Creating teams might require complex payloads depending on API, doing raw SQL for Speed/Reliability
    team_a_id = str(uuid.uuid4())
    team_b_id = str(uuid.uuid4())
    division_id = str(uuid.uuid4())
    field_id = str(uuid.uuid4())
    game_id = str(uuid.uuid4())
    
    with db_connection.cursor() as cur:
        # Create a division
        cur.execute("INSERT INTO division_pools (id, name, type, event_id) VALUES (%s, %s, %s, %s)", 
                    (division_id, "Open Division", "pool", event_id))
        
        # Create an event location
        cur.execute("INSERT INTO fields (id, name, event_id) VALUES (%s, %s, %s)",
                    (field_id, "Field 1", event_id))
        
        # Create teams
        cur.execute("INSERT INTO teams (id, name, event_id, division_id) VALUES (%s, %s, %s, %s)",
                    (team_a_id, "Team A", event_id, division_id))
        cur.execute("INSERT INTO teams (id, name, event_id, division_id) VALUES (%s, %s, %s, %s)",
                    (team_b_id, "Team B", event_id, division_id))
        
        # Create a game
        # status: scheduled
        now = datetime.datetime.now(datetime.timezone.utc).isoformat()
        cur.execute("""
            INSERT INTO games (id, name, scheduled_time, allocated_time_minutes, status, home_team_score, away_team_score, version, home_team_id, away_team_id, division_pool_id, field_location_id) 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        """, (game_id, "Team A vs Team B", now, 60, "scheduled", 0, 0, 1, team_a_id, team_b_id, division_id, field_id))

    # A game was created, let's submit a spirit score for Team A from Team B (scored_by_team_id = Team B, team_id = Team A)
    # Submitted by Scorekeeper
    spirit_a_payload = {
        "scored_by_team_id": team_b_id,
        "team_id": team_a_id,
        "rules_knowledge": 3,
        "fouls_body_contact": 2,
        "fair_mindedness": 4,
        "attitude": 3,
        "communication": 4,
        "comments": "Great game team A!"
    }
    
    resp_spirit_a = scorekeeper_client.post(f"/games/{game_id}/spirit", json=spirit_a_payload)
    assert resp_spirit_a.status_code == 201, f"Failed to submit Spirit Score A: {resp_spirit_a.text}"

    # Verify score
    spirit_id = resp_spirit_a.json().get("id")
    assert spirit_id is not None
    assert resp_spirit_a.json().get("total_score") == 16 # 3+2+4+3+4

    # Check Spirit average API for Team A
    resp_avg = scorekeeper_client.get(f"/teams/{team_a_id}/spirit-average")
    assert resp_avg.status_code == 200, f"Average fetching failed: {resp_avg.text}"
    avg_data = resp_avg.json()
    assert avg_data["games_played"] == 1
    assert avg_data["average_total"] == 16.0

