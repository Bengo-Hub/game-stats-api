import json
import os

file_path = r'd:\Projects\Codevertex\mosuon\game-stats\game-stats-api\scripts\fixtures\games_game.json'

if not os.path.exists(file_path):
    print(f"File not found: {file_path}")
    exit(1)

with open(file_path, 'r', encoding='utf-8') as f:
    data = json.load(f)

for item in data:
    fields = item.get('fields', {})
    
    # Rename fields
    if 'start_time' in fields:
        fields['date'] = fields.pop('start_time')
    if 'team1_score' in fields:
        fields['home_team_score'] = fields.pop('team1_score')
    if 'team2_score' in fields:
        fields['away_team_score'] = fields.pop('team2_score')
    if 'pool' in fields:
        fields['division_pool'] = fields.pop('pool')
    if 'team1' in fields:
        fields['home_team'] = fields.pop('team1')
    if 'team2' in fields:
        fields['away_team'] = fields.pop('team2')

with open(file_path, 'w', encoding='utf-8') as f:
    json.dump(data, f, indent=2)

print(f"Successfully updated {file_path}")
