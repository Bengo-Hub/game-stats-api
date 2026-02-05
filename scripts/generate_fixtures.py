import os
import json

SQL_FILE = os.path.join(os.getcwd(), 'data1.sql')
OUT_DIR = os.path.join(os.getcwd(), 'scripts', 'fixtures')

TABLE_TO_FIXTURE = {
    '_core_location': ('core.location', {
        'id': 'pk', 'name': 'name', 'address': 'address', 'city': 'city', 'country': 'country'
    }),
    '_core_divisionpool': ('games.divisionpool', {'id': 'pk', 'name': 'name', 'description': 'description'}),
    '_core_field': ('games.field', {'id': 'pk', 'name': 'name', 'capacity': 'capacity', 'surface_type': 'surface_type', 'location_id': 'location'}),
    '_core_gameround': ('games.gameround', {'id': 'pk', 'name': 'name', 'start_date': 'start_date', 'end_date': 'end_date'}),
    '_core_team': ('games.team', {'id': 'pk', 'name': 'name', 'initial_seed': 'initial_seed', 'origin_id': 'origin'}),
    '_core_player': ('games.player', {'id': 'pk', 'name': 'name', 'spirit_award_nominations': 'spirit_award_nominations', 'mvp_nominations': 'mvp_nominations', 'team_id': 'team', 'gender': 'gender'}),
    '_core_game': ('games.game', {'id': 'pk', 'date': 'start_time', 'home_team_score': 'team1_score', 'away_team_score': 'team2_score', 'division_pool_id': 'pool', 'field_id': 'field', 'away_team_id': 'team2', 'home_team_id': 'team1', 'name': 'name', 'game_round_id': 'game_round'}),
    '_core_scoring': ('games.scoring', {'id': 'pk', 'goals': 'goals', 'assists': 'assists', 'game_id': 'game', 'player_id': 'player'}),
    '_core_spiritscore': ('games.spiritscore', {'id': 'pk', 'rules_knowledge': 'rules_knowledge', 'fouls_body_contact': 'fouls_body_contact', 'fair_mindedness': 'fair_mindedness', 'attitude': 'attitude', 'communication': 'communication', 'game_id': 'game', 'scored_by_id': 'scored_by', 'team_id': 'team', 'mvp_female_nomination_id': 'mvp_female_nomination', 'mvp_male_nomination_id': 'mvp_male_nomination', 'spirit_female_nomination_id': 'spirit_female_nomination', 'spirit_male_nomination_id': 'spirit_male_nomination'}),
    '_authman_user': ('authman.user', {'id': 'pk', 'password': 'password', 'last_login': 'last_login', 'is_superuser': 'is_superuser', 'username': 'username', 'first_name': 'first_name', 'last_name': 'last_name', 'email': 'email', 'is_staff': 'is_staff', 'is_active': 'is_active', 'date_joined': 'date_joined', 'role': 'role', 'team_id': 'team'}),
}

if __name__ == '__main__':
    os.makedirs(OUT_DIR, exist_ok=True)
    fixtures = {v[0]: [] for v in TABLE_TO_FIXTURE.values()}

    with open(SQL_FILE, 'r', encoding='utf-8') as f:
        line = f.readline()
        while line:
            if line.startswith('COPY public.'):
                header = line.strip()
                parts = header.split()
                table = parts[1].split('.')[-1]
                cols_part = header[header.find('(')+1:header.find(')')]
                cols = [c.strip() for c in cols_part.split(',')]
                rows = []
                while True:
                    line = f.readline()
                    if not line:
                        break
                    if line.strip() == '\\.':
                        break
                    rows.append(line.rstrip('\n'))

                if table in TABLE_TO_FIXTURE:
                    model_label, colmap = TABLE_TO_FIXTURE[table]
                    for r in rows:
                        values = r.split('\t')
                        data = dict(zip(cols, values))
                        pk = int(data.get('id')) if data.get('id') else None
                        fields = {}
                        for sql_col, model_col in colmap.items():
                            if sql_col == 'id':
                                continue
                            val = data.get(sql_col)
                            if val == '' or val is None:
                                continue
                            # numeric conversions for obvious fields
                            if model_col in ('capacity', 'initial_seed', 'team1_score', 'team2_score', 'goals', 'assists'):
                                try:
                                    val = int(val)
                                except ValueError:
                                    pass
                            fields[model_col] = (int(val) if (model_col.endswith('_id') or model_col in ('team','game','player','pool','field','team1','team2','scored_by','mvp_female_nomination','mvp_male_nomination','spirit_female_nomination','spirit_male_nomination')) and str(val).isdigit() else val)

                        fixtures[model_label].append({'model': model_label, 'pk': pk, 'fields': fields})
            else:
                line = f.readline()

    # Write out fixture files
    for model_label, items in fixtures.items():
        if not items:
            continue
        safe_name = model_label.replace('.', '_')
        out_file = os.path.join(OUT_DIR, f'{safe_name}.json')
        with open(out_file, 'w', encoding='utf-8') as of:
            json.dump(items, of, default=str, indent=2)
        print(f'Wrote {len(items)} records to {out_file}')

    print('Fixture generation complete. Use `python manage.py loaddata scripts/fixtures/<file>.json` to import.')