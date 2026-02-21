import pytest
import io

def test_mass_upload_players(admin_client):
    # 1. Create a dummy CSV file in memory
    # Format: Name, Gender, JerseyNumber
    csv_content = "Name,Gender,JerseyNumber\nBulk Player 1,M,10\nBulk Player 2,F,20\nBulk Player 3,X,30"
    file = io.BytesIO(csv_content.encode('utf-8'))
    
    # 2. Get a list of teams to pick one
    # Note: Using /public/teams since it's already there and works
    teams_resp = admin_client.get("/public/teams")
    assert teams_resp.status_code == 200
    teams = teams_resp.json()
    assert len(teams) > 0, "No teams found to upload players to. Please seed the database first."
    team_id = teams[0]['id']
    
    # 3. Post the CSV to the bulk upload endpoint
    # Expected endpoint: POST /api/v1/teams/{id}/players/upload
    files = {'file': ('players.csv', file, 'text/csv')}
    resp = admin_client.post(f"/teams/{team_id}/players/upload", files=files)
    
    # 4. Verify response
    assert resp.status_code == 200
    result = resp.json()
    assert result['count'] == 3
    assert len(result.get('errors', [])) == 0
    
    # 5. Verify players exist in the roster
    # Note: Use /public/teams/{id} which includes players in its response
    team_detail_resp = admin_client.get(f"/public/teams/{team_id}")
    assert team_detail_resp.status_code == 200
    team_data = team_detail_resp.json()
    players = team_data.get('players', [])
    player_names = [p['name'] for p in players]
    
    assert "Bulk Player 1" in player_names
    assert "Bulk Player 2" in player_names
    assert "Bulk Player 3" in player_names
    
    # Verify jersey numbers and genders
    for player in players:
        if player['name'] == "Bulk Player 1":
            assert player['gender'] == "M"
            assert player['jerseyNumber'] == 10
        elif player['name'] == "Bulk Player 2":
            assert player['gender'] == "F"
            assert player['jerseyNumber'] == 20
        elif player['name'] == "Bulk Player 3":
            assert player['gender'] == "X"
            assert player['jerseyNumber'] == 30
