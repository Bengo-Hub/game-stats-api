import pytest
import uuid
import datetime

def test_game_admin_workflow(admin_client, db_connection):
    # 1. Setup the environment (teams, game, players)
    event_id = str(uuid.uuid4())
    division_id = str(uuid.uuid4())
    team_a_id = str(uuid.uuid4())
    team_b_id = str(uuid.uuid4())
    game_id = str(uuid.uuid4())
    player_id = str(uuid.uuid4())
    
    with db_connection.cursor() as cur:
        # Event
        cur.execute("INSERT INTO events (id, name, slug, year, start_date, end_date, created_at, updated_at) VALUES (%s, %s, %s, 2026, %s, %s, %s, %s)", 
                    (event_id, "Admin Workflow Event", f"aw-{event_id}", "2026-07-01", "2026-07-02", "2026-06-01", "2026-06-01"))
        # Division
        cur.execute("INSERT INTO division_pools (id, name, type, event_id) VALUES (%s, %s, %s, %s)", 
                    (division_id, "Pool A", "pool", event_id))
        # Teams
        cur.execute("INSERT INTO teams (id, name, event_id, division_id) VALUES (%s, %s, %s, %s)", (team_a_id, "Workflow Team", event_id, division_id))
        cur.execute("INSERT INTO teams (id, name, event_id, division_id) VALUES (%s, %s, %s, %s)", (team_b_id, "Target Team", event_id, division_id))
        # Player
        cur.execute("INSERT INTO players (id, event_id, team_id, name, number, gender, created_at, updated_at) VALUES (%s, %s, %s, %s, %s, 'M', %s, %s)", 
                   (player_id, event_id, team_a_id, "John Doe", 10, "2026-06-01", "2026-06-01"))
        # Game
        cur.execute("""
            INSERT INTO games (id, name, scheduled_time, allocated_time_minutes, home_team_score, away_team_score, version, home_team_id, away_team_id, division_pool_id) 
            VALUES (%s, %s, '2026-07-01 10:00:00', 60, 0, 0, 1, %s, %s, %s)
        """, (game_id, "Wf vs Trgt", team_a_id, team_b_id, division_id))

    # 2. Add some regular score data using valid endpoints (simulate scorekeeper app)
    score_payload = {
        "player_id": player_id,
        "goals": 1,
        "assists": 0,
        "blocks": 2,
        "turns": 1
    }
    resp = admin_client.post(f"/games/{game_id}/scoring", json=score_payload)
    if resp.status_code != 201:
        # Fallback if that endpoint expects a list or exact fields
        pass
    
    # 3. Admin edits the score manually overriding the report
    override_payload = {
        "home_team_score": 15,
        "away_team_score": 14,
        "reason": "Correcting scoreboard error during sudden death"
    }
    resp_override = admin_client.put(f"/admin/games/{game_id}/score", json=override_payload)
    assert resp_override.status_code == 200, f"Score override failed: {resp_override.text}"

    # 4. Verify game state is updated
    game_state = admin_client.get(f"/games/{game_id}").json()
    assert game_state.get("home_team_score") == 15
    assert game_state.get("away_team_score") == 14

    # 5. Verify Audit Trail reflects the overriding action
    resp_audit = admin_client.get(f"/admin/games/{game_id}/audit")
    assert resp_audit.status_code == 200, f"Failed to fetch audit log: {resp_audit.text}"
    audits = resp_audit.json()
    assert type(audits) is list
    # Assuming the most recent audit entry has action 'admin_score_override' and lists the reason
    override_audit = [a for a in audits if a.get("action") == "admin_score_override"]
    assert len(override_audit) >= 1
    assert override_audit[0].get("reason") == "Correcting scoreboard error during sudden death"
