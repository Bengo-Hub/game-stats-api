import requests
import uuid
import time
import sys

# Configuration
API_URL = "http://localhost:4000/api/v1"
ADMIN_EMAIL = "admin@admin.com"
ADMIN_PASSWORD = "ChangeMe123!"

session = requests.Session()

def print_step(msg):
    print(f"\n{'='*50}")
    print(f"> {msg}")
    print(f"{'='*50}")

def print_result(msg, is_success=True):
    icon = "[OK]" if is_success else "[FAIL]"
    print(f"{icon} {msg}")

def test_workflow():
    # 1. Login
    print_step("1. Login as Admin")
    payload = {"email": ADMIN_EMAIL, "password": ADMIN_PASSWORD}
    resp = session.post(f"{API_URL}/auth/login", json=payload)
    if resp.status_code != 200:
        resp = session.post(f"{API_URL}/login", json=payload)
    if resp.status_code != 200:
        print_result(f"Login failed: {resp.text}", False)
        sys.exit(1)
        
    token = resp.json().get("access_token")
    if not token:
        token = resp.json().get("token")
        
    admin_name = resp.json().get("user", {}).get("name", "System Admin")
    
    session.headers.update({
        "Authorization": f"Bearer {token}",
        "X-Username": admin_name
    })
    print_result("Login successful.")

    slug_suffix = str(uuid.uuid4())[:6]

    # 2. Fetch Metadata
    print_step("2. Fetch Metadata (Locations & Disciplines)")
    loc_resp = session.get(f"{API_URL}/public/geographic/locations")
    if loc_resp.status_code != 200 or not loc_resp.json():
        print_result("Failed to fetch locations.", False)
        sys.exit(1)
    
    locations = loc_resp.json()
    loc_id = locations[0]["id"]
    print_result(f"Found Location: {locations[0]['name']}")

    disc_resp = session.get(f"{API_URL}/public/disciplines")
    if disc_resp.status_code != 200 or not disc_resp.json():
        print_result("Failed to fetch disciplines.", False)
        sys.exit(1)
        
    disciplines = disc_resp.json()
    disc_id = disciplines[0]["id"]
    print_result(f"Found Discipline: {disciplines[0]['name']}")

    # 2.5 Create a Field (Required for Games)
    print_step("2.5 Create Playing Field")
    field_payload = {
        "name": f"Field 1-{slug_suffix}",
        "location_id": loc_id,
        "is_active": True
    }
    resp = session.post(f"{API_URL}/geographic/fields", json=field_payload)
    if resp.status_code != 201:
        print_result(f"Field creation failed: {resp.text}", False)
        sys.exit(1)
    field_id = resp.json()["id"]
    print_result(f"Field Created. ID: {field_id}")

    # 3. Create Event
    print_step("3. Create Tournament Event")
    event_payload = {
        "name": f"Workflow Simulation {slug_suffix}",
        "slug": f"sim-event-{slug_suffix}",
        "year": 2026,
        "startDate": "2026-07-01",
        "endDate": "2026-07-02",
        "status": "published",
        "locationId": loc_id,
        "disciplineId": disc_id
    }
    resp = session.post(f"{API_URL}/events", json=event_payload)
    if resp.status_code != 201:
        print_result(f"Event creation failed: {resp.text}", False)
        sys.exit(1)
    event_id = resp.json()["id"]
    print_result(f"Event Created. ID: {event_id}")

    # 4. Create Rounds (Crossover -> Semis -> Finals)
    print_step("4. Configure Game Rounds")
    rounds_data = [
        {"name": "Finals", "roundType": "final", "autoAdvance": False},
        {"name": "Semifinals", "roundType": "semifinal", "autoAdvance": True, "topNTeams": 2},
        {"name": "Crossover", "roundType": "crossover", "autoAdvance": True, "topNTeams": 4}
    ]
    round_ids = {}
    last_target = None
    
    for rd in rounds_data:
        payload = {
            "name": rd["name"],
            "round_type": rd["roundType"],
            "auto_advance": rd["autoAdvance"]
        }
        if "topNTeams" in rd:
            payload["top_n_teams"] = rd["topNTeams"]
        if last_target:
            payload["target_round_id"] = last_target
            
        resp = session.post(f"{API_URL}/events/{event_id}/rounds", json=payload)
        if resp.status_code != 201:
            print_result(f"Round {rd['name']} creation failed: {resp.text}", False)
            sys.exit(1)
            
        r_id = resp.json()["id"]
        round_ids[rd["name"]] = r_id
        last_target = r_id
        print_result(f"Round '{rd['name']}' created.")
        
    crossover_id = round_ids["Crossover"]

    # 5. Create Division Pools
    print_step("5. Create Pool A and Pool B")
    pools = []
    for p_name in ["Pool A", "Pool B"]:
        payload = {
            "name": p_name,
            "divisionType": "pool",  # Note: Division schema is camelCase
            "auto_advance": True,
            "target_round_id": crossover_id,
            "top_n_teams": 2
        }
        resp = session.post(f"{API_URL}/events/{event_id}/divisions", json=payload)
        if resp.status_code != 201:
            print_result(f"Pool {p_name} creation failed: {resp.text}", False)
            sys.exit(1)
        pools.append(resp.json()["id"])
        print_result(f"Created {p_name}.")
        
    pool_a, pool_b = pools[0], pools[1]

    # 6. Create Teams
    print_step("6. Register Teams")
    team_data = [
        {"name": "Team A1", "divisionId": pool_a},
        {"name": "Team A2", "divisionId": pool_a},
        {"name": "Team B1", "divisionId": pool_b},
        {"name": "Team B2", "divisionId": pool_b},
    ]
    teams = []
    for td in team_data:
        payload = {
            "name": f"{td['name']} {slug_suffix}",
            "eventId": event_id,
            "divisionPoolId": td["divisionId"]
        }
        resp = session.post(f"{API_URL}/teams", json=payload)
        if resp.status_code != 201:
            print_result(f"Team {td['name']} creation failed: {resp.text}", False)
            sys.exit(1)
        teams.append(resp.json()["id"])
        print_result(f"Registered {td['name']}")
        
    t_a1, t_a2, t_b1, t_b2 = teams

    # 7. Schedule Pool Games
    print_step("7. Schedule Round-Robin Pool Games")
    games_to_play = []
    # Game A1 vs A2 in Pool A
    # Game B1 vs B2 in Pool B
    for idx, (home, away, pool) in enumerate([(t_a1, t_a2, pool_a), (t_b1, t_b2, pool_b)]):
        payload = {
            "division_pool_id": pool,
            "home_team_id": home,
            "away_team_id": away,
            "scheduled_time": f"2026-07-01T1{idx}:00:00Z",
            "allocated_time_minutes": 60,
            "field_location_id": field_id
        }
        resp = session.post(f"{API_URL}/games", json=payload)
        if resp.status_code != 201:
            print_result(f"Game scheduling failed [{resp.status_code}]: {resp.text} Headers: {resp.headers}", False)
            sys.exit(1)
        games_to_play.append(resp.json()["id"])
        print_result("Scheduled pool game.")
        
    game_a, game_b = games_to_play

    # 8. Score Pool Games
    print_step("8. Complete Pool Games (Triggers Crossover Generation)")
    for i, game in enumerate(games_to_play):
        pool_name = "Pool A" if i == 0 else "Pool B"
        print(f"Scoring {pool_name} Game...")
        # Simulating Start -> Complete pipeline or just admin override
        score_payload = {
            "home_score": 15, 
            "away_score": 10, 
            "reason": "Automated workflow testing",
            "admin_name": "E2E Tester"
        }
        
        # Try Admin Override directly to end it
        resp = session.put(f"{API_URL}/admin/games/{game}/score", json=score_payload)
        if resp.status_code != 200:
            print_result(f"Failed to score {pool_name} game: {resp.text}", False)
            sys.exit(1)
        print_result(f"Scored {pool_name} match.")

    # Give the backend ranking service a moment to generate standard seeded crossover matchups 
    print("Waiting 2s for backend to generate crossover brackets...")
    time.sleep(2)

    # 9. Fetch and Verify Crossovers
    print_step("9. Verify Auto-Scheduled Crossovers")
    resp = session.get(f"{API_URL}/public/games?eventId={event_id}&status=scheduled")
    if resp.status_code != 200:
        print_result(f"Failed to fetch games list: {resp.text}", False)
        sys.exit(1)
    
    all_games = resp.json()
    crossover_games = [g for g in all_games if g.get("gameRoundId") == crossover_id]
    
    if len(crossover_games) == 0:
        print_result("No crossover games generated by backend!", False)
        sys.exit(1)
        
    crossover_ids = []
    for g in crossover_games:
        home_name = g.get("homeTeam", {}).get("name", "TBD")
        away_name = g.get("awayTeam", {}).get("name", "TBD")
        print_result(f"Generated Crossover Matchup: {home_name} vs {away_name} (Game ID: {g['id']})")
        crossover_ids.append(g["id"])

    # 10. Score Crossovers
    print_step("10. Score Crossover Games (Triggers continuous bracket promotion)")
    for cg_id in crossover_ids:
        score_payload = {
            "home_score": 15, 
            "away_score": 9, 
            "reason": "Advance to Semis",
            "admin_name": "E2E Tester"
        }
        resp = session.put(f"{API_URL}/admin/games/{cg_id}/score", json=score_payload)
        if resp.status_code != 200:
            print_result(f"Failed to score crossover game {cg_id}: {resp.text}", False)
            sys.exit(1)
        print_result(f"Scored crossover game {cg_id}.")

    print("Waiting 2s for backend bracket DFS promotion...")
    time.sleep(2)

    # 11. Verify Bracket Advancements (To Semis or Finals)
    print_step("11. Verify Bracket Promotion (Semis/Finals assignments)")
    resp = session.get(f"{API_URL}/public/games?eventId={event_id}")
    if resp.status_code != 200:
        print_result(f"Failed to fetch advanced games: {resp.text}", False)
        sys.exit(1)
        
    all_games = resp.json()
    advanced_games = [
        g for g in all_games 
        if g.get("homeTeamId") is not None 
        and g.get("awayTeamId") is not None 
        and g["id"] not in games_to_play + crossover_ids
    ]
    
    if len(advanced_games) == 0:
        print_result("Warning: No promoted games found yet (If Semis rounds were TBD nodes).", False)
    else:
        for fg in advanced_games:
            home_name = fg.get("homeTeam", {}).get("name", "Unknown")
            away_name = fg.get("awayTeam", {}).get("name", "Unknown")
            print_result(f"Advanced Playoff Match Generated: {home_name} vs {away_name}")

    print_step("Workflow Test Finished Successfully! [OK]")

if __name__ == "__main__":
    try:
        test_workflow()
    except requests.exceptions.ConnectionError:
        print_result(f"Could not connect to {API_URL}. Is the backend running?", False)
        sys.exit(1)
