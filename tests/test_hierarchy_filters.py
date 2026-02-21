import pytest

def test_player_leaderboard_sorting(api_client):
    # 1. Generate 10 players with various goals and assists
    # 2. Query /api/v1/players/leaderboard?sort=total
    # 3. Verify the list is strictly ordered by (goals + assists) descending
    # 4. Apply a gender filter (?gender=female) and verify only matching players are returned
    pass
