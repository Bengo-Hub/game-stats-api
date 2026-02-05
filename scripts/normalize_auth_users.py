import json
from pathlib import Path

IN = Path('scripts/fixtures/authman_user.json')
BAK = Path('scripts/fixtures/authman_user.json.bak')

with IN.open('r', encoding='utf-8') as f:
    data = json.load(f)

# make a backup
BAK.write_text(json.dumps(data, indent=2, ensure_ascii=False), encoding='utf-8')

for obj in data:
    if obj.get('model') != 'authman.user':
        continue
    fields = obj['fields']
    # Normalize boolean strings to booleans
    for b in ('is_superuser', 'is_staff', 'is_active'):
        if b in fields and isinstance(fields[b], str):
            fields[b] = True if fields[b] in ('1', 'true', 'True') else False
    # Normalize datetimes to ISO with Z if naive
    for dt_field in ('last_login', 'date_joined'):
        val = fields.get(dt_field)
        if isinstance(val, str) and val:
            # If already contains timezone or 'T' + offset assume ok
            if 'T' not in val or ('+' not in val and 'Z' not in val and '-' not in val[11:]):
                # replace first space with 'T' and append 'Z' if no timezone info
                new = val.replace(' ', 'T')
                if 'Z' not in new and ('+' not in new and '-' not in new[11:]):
                    new = new + 'Z'
                fields[dt_field] = new
    # Add groups from role
    groups = fields.get('groups') or []
    if fields.get('is_superuser'):
        if 1 not in groups:
            groups.append(1)
    role = fields.get('role')
    if role == 'team_manager' and 2 not in groups:
        groups.append(2)
    if groups:
        fields['groups'] = groups

with IN.open('w', encoding='utf-8') as f:
    json.dump(data, f, indent=2, ensure_ascii=False)

print('Normalized', IN, '-> backup at', BAK)
