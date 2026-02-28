import requests
import sys
import time
import uuid
import random

# Configuration
API_URL = "http://localhost:4000/api/v1"
ADMIN_EMAIL = "admin@admin.com"
ADMIN_PASSWORD = "ChangeMe123!"
slug_suffix = str(uuid.uuid4())[:8]
random_day = random.randint(1, 28)

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
        
    data = resp.json()
    token = data.get("access_token") or data.get("token")
    user = data.get("user", {})
    admin_name = user.get("name") or user.get("username") or "System Admin"
    
    session.headers.update({
        "Authorization": f"Bearer {token}",
        "X-Username": admin_name
    })
    print_result(f"Login successful as {admin_name}.")

    slug_suffix = str(uuid.uuid4())[:6]

    # 2. Setup Tournament Structure
    print_step("2. Setup Event, Rounds, and Pools")
    
    # Get Location & Discipline
    loc_resp = session.get(f"{API_URL}/public/geographic/locations")
    loc_id = loc_resp.json()[0]["id"]
    disc_resp = session.get(f"{API_URL}/public/disciplines")
    disc_id = disc_resp.json()[0]["id"]
    
    # Get Field
    field_resp = session.get(f"{API_URL}/public/geographic/fields")
    field_id = field_resp.json()[0]["id"]

    # Create Event
    event_payload = {
        "name": f"Spirit & Crossover Test {slug_suffix}",
        "slug": f"test-event-{slug_suffix}",
        "year": 2026,
        "startDate": "2026-07-01",
        "endDate": "2026-07-02",
        "status": "published",
        "locationId": loc_id,
        "disciplineId": disc_id
    }
    event_id = session.post(f"{API_URL}/events", json=event_payload).json()["id"]
    print_result(f"Event Created: {event_id}")

    # Create Rounds (Finals <- Crossover)
    finals_payload = {"name": "Finals", "round_type": "final", "auto_advance": False}
    finals_id = session.post(f"{API_URL}/events/{event_id}/rounds", json=finals_payload).json()["id"]

    crossover_payload = {
        "name": "Crossover", 
        "round_type": "crossover", 
        "auto_advance": True, 
        "target_round_id": finals_id,
        "top_n_teams": 2
    }
    crossover_id = session.post(f"{API_URL}/events/{event_id}/rounds", json=crossover_payload).json()["id"]
    print_result("Rounds Created (Crossover -> Finals)")

    # Create Pools (Pool A & Pool B -> Crossover)
    pool_a_id = session.post(f"{API_URL}/events/{event_id}/divisions", json={
        "name": "Pool A", "divisionType": "pool", "auto_advance": True, "target_round_id": crossover_id, "top_n_teams": 1
    }).json()["id"]
    pool_b_id = session.post(f"{API_URL}/events/{event_id}/divisions", json={
        "name": "Pool B", "divisionType": "pool", "auto_advance": True, "target_round_id": crossover_id, "top_n_teams": 1
    }).json()["id"]
    print_result("Pools A & B Created")

    # 3. Create Teams and Players
    print_step("3. Register Teams and Players")
    teams = []
    for p_id, t_name in [(pool_a_id, "Titans"), (pool_b_id, "Giants")]:
        t_id = session.post(f"{API_URL}/teams", json={
            "name": f"{t_name} {slug_suffix}", "eventId": event_id, "divisionPoolId": p_id
        }).json()["id"]
        
        # Add 1 Male and 1 Female player to each team
        players = []
        for gender, p_name in [("M", "M-Player"), ("F", "F-Player")]:
            p_resp = session.post(f"{API_URL}/teams/{t_id}/players", json={
                "name": f"{p_name}-{t_name}", "gender": gender, "team_id": t_id
            })
            players.append(p_resp.json()["id"])
        
        teams.append({"id": t_id, "name": t_name, "players": players})
        print_result(f"Team {t_name} registered with players.")

    # 4. Schedule and Play Pool Game (Dummy opponent needed for pool play usually, but let's assume 1-team pools for simple trigger or add A2/B2)
    # To keep it standard, let's add A2/B2
    print_step("4. Round Robin Pool Play")
    titans_game_id = None
    for i, team_info in enumerate(teams):
        print_result(f"Scheduling pool game for {team_info['name']}...")
        # Opponent is the other team
        opp = teams[1] if team_info["name"] == "Titans" else teams[0]
        opp_id = opp["id"]
        pool_id = pool_a_id if team_info["name"] == "Titans" else pool_b_id
        
        # Schedule Game (Vary time to avoid field conflict)
        game_payload = {
            "division_pool_id": pool_id,
            "home_team_id": team_info["id"],
            "away_team_id": opp_id,
            "scheduled_time": f"2030-07-{random_day:02d}T{10+i}:00:00Z",
            "allocated_time_minutes": 60,
            "field_location_id": field_id
        }
        resp = session.post(f"{API_URL}/games", json=game_payload)
        if resp.status_code != 201:
            print_result(f"Game scheduling failed: {resp.status_code} - {resp.text}", False)
            sys.exit(1)
        game_id = resp.json()["id"]
        if team_info["name"] == "Titans":
            titans_game_id = game_id
        
        # Score Game (Admin Override)
        resp = session.put(f"{API_URL}/admin/games/{game_id}/score", json={
            "home_score": 15, "away_score": 5, "reason": "Advance Test"
        })
        if resp.status_code != 200:
            print_result(f"Score recording failed: {resp.status_code} - {resp.text}", False)
            sys.exit(1)
            
        # Complete Game (Sets status to completed)
        resp = session.post(f"{API_URL}/games/{game_id}/complete")
        if resp.status_code != 200:
            print_result(f"Game completion failed: {resp.status_code} - {resp.text}", False)
            sys.exit(1)
            
        print_result(f"Pool game for {team_info['name']} completed.")

    time.sleep(2) # Backend processing

    # 5. Verify Crossover and Submit Spirit Score
    print_step("5. Dual-Gender Spirit Score Submission")
    
    # Get Giants Team for nominations
    giants = next(t for t in teams if t["name"] == "Giants")
    away_team_id = giants["id"]
    titans_id = next(t for t in teams if t["name"] == "Titans")["id"]
    game_id = titans_game_id
    
    # Submit Spirit Score (Titans rating Giants)
    # scored_by_team_id is Titans, team_id is Giants
    spirit_payload = {
        "scored_by_team_id": pool_game["homeTeam"]["id"],
        "team_id": away_team_id,
        "rules_knowledge": 4,
        "fouls_body_contact": 4,
        "fair_mindedness": 4,
        "attitude": 4,
        "communication": 4,
        "comments": "Great game!",
        "mvp_male_id": giants["players"][0], # Male Giants
        "mvp_female_id": giants["players"][1], # Female Giants
        "spirit_male_id": giants["players"][0], 
        "spirit_female_id": giants["players"][1]
    }
    
    resp = session.post(f"{API_URL}/games/{game_id}/spirit", json=spirit_payload)
    if resp.status_code == 201:
        print_result("Spirit Score with dual-gender nominations submitted successfully.")
        
        # Verify the summary response
        data = resp.json()
        required_fields = ["mvpMaleNomination", "mvpFemaleNomination", "spiritMaleNomination", "spiritFemaleNomination"]
        missing = [f for f in required_fields if f not in data]
        if not missing:
            print_result("All 4 nomination fields found in the response.")
            if data["mvpMaleNomination"]["name"].startswith("M-Player"):
                print_result(f"Nomination data correct: {data['mvpMaleNomination']['name']}")
            else:
                print_result(f"Nomination name mismatch: {data['mvpMaleNomination']['name']}", False)
        else:
            print_result(f"Missing nomination fields in response: {missing}", False)
    else:
        print_result(f"Spirit Score submission failed: {resp.text}", False)

    # 6. Verify Crossover Matchup
    print_step("6. Verify Crossover Generation")
    resp = session.get(f"{API_URL}/public/games?eventId={event_id}")
    games = resp.json()
    crossover_games = [g for g in games if g.get("gameRoundId") == crossover_id]
    if crossover_games:
        print_result(f"Found {len(crossover_games)} auto-generated crossover games.")
        for g in crossover_games:
            home = g.get("homeTeam", {}).get("name", "TBD")
            away = g.get("awayTeam", {}).get("name", "TBD")
            print_result(f"Matchup: {home} vs {away}")
    else:
        print_result("No crossover games found. Check backend logs.", False)

    print_step("E2E Test Completed")

if __name__ == "__main__":
    test_workflow()
