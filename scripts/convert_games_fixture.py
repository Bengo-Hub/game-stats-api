import json
from datetime import datetime

in_path = 'scripts/fixtures/games_game.json'
out_path = 'scripts/fixtures/games_game_fixed.json'

with open(in_path, 'r', encoding='utf-8') as f:
    data = json.load(f)

new_data = []
for obj in data:
    if obj.get('model') != 'games.game':
        new_data.append(obj)
        continue
    fields = obj['fields']
    # Normalize keys
    if 'start_time' in fields:
        fields['date'] = fields.pop('start_time')
    if 'team1' in fields:
        fields['home_team'] = fields.pop('team1')
    if 'team2' in fields:
        fields['away_team'] = fields.pop('team2')
    if 'team1_score' in fields:
        fields['home_team_score'] = fields.pop('team1_score')
    if 'team2_score' in fields:
        fields['away_team_score'] = fields.pop('team2_score')
    if 'pool' in fields:
        fields['division_pool'] = fields.pop('pool')
    # Ensure game_round is int if present
    if 'game_round' in fields:
        try:
            fields['game_round'] = int(str(fields['game_round']))
        except Exception:
            pass
    # Ensure date format is ISO; convert if space separated
    if 'date' in fields:
        # if already iso-like skip
        if 'T' not in fields['date']:
            fields['date'] = fields['date'].replace(' ', 'T')
    # Add location and status if missing
    if 'location' not in fields:
        fields['location'] = 1
    if 'status' not in fields:
        fields['status'] = 'completed'

    obj['fields'] = fields
    new_data.append(obj)

with open(out_path, 'w', encoding='utf-8') as f:
    json.dump(new_data, f, indent=2, ensure_ascii=False)

print('Converted', len(new_data), 'objects')