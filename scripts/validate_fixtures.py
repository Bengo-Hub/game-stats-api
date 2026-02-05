import json
import re
from pathlib import Path
import sys
import os

# Ensure Django project on path
PROJECT_ROOT = Path(__file__).resolve().parents[1]
if str(PROJECT_ROOT) not in sys.path:
    sys.path.insert(0, str(PROJECT_ROOT))

from django.apps import apps
import django

# Quick validator/fixer for fixtures

def is_tz_aware(dt_str: str) -> bool:
    return 'T' in dt_str and (dt_str.endswith('Z') or re.search(r'[+-]\d\d:\d\d$', dt_str))


def validate_file(path: Path):
    print(f"Validating {path}")
    try:
        data = json.loads(path.read_text(encoding='utf-8'))
    except Exception as exc:
        print('  ✖ JSON parse error:', exc)
        return False

    modified = False
    for obj in data:
        model_label = obj.get('model')
        if not model_label:
            print('  ⚠ missing model label in object', obj)
            continue
        app_label, model_name = model_label.split('.')
        try:
            apps.get_model(app_label, model_name)
        except LookupError:
            print(f'  ⚠ Model not found: {app_label}.{model_name}')
        fields = obj.get('fields', {})
        for k, v in list(fields.items()):
            if isinstance(v, str) and v:
                # datetime-like quick check
                if re.match(r'\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}', v):
                    new = v.replace(' ', 'T') + 'Z'
                    fields[k] = new
                    modified = True
                    print(f"  ✓ Fixed naive dt on {model_label} pk={obj.get('pk')} field={k}")
                elif re.match(r'\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$', v):
                    fields[k] = v + 'Z'
                    modified = True
                    print(f"  ✓ Added Z timezone on {model_label} pk={obj.get('pk')} field={k}")
            # boolean normalization
            if k in ('is_superuser', 'is_staff', 'is_active') and isinstance(v, str):
                if v in ('1', '0'):
                    fields[k] = True if v == '1' else False
                    modified = True
                    print(f"  ✓ Normalized boolean {k} on {model_label} pk={obj.get('pk')}")
    if modified:
        Path(str(path) + '.bak').write_text(path.read_text(encoding='utf-8'), encoding='utf-8')
        path.write_text(json.dumps(data, indent=2, ensure_ascii=False), encoding='utf-8')
        print('  → Wrote fixes and created .bak')
    return True


if __name__ == '__main__':
    os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'DigiGameStats.settings')
    django.setup()

    fixtures_dir = Path('scripts/fixtures')
    ok = True
    for f in fixtures_dir.glob('*.json'):
        ok = validate_file(f) and ok
    print('\nValidation complete. OK=' + str(ok))
    if not ok:
        raise SystemExit(1)
