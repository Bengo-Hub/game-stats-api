import os
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[1]
if str(PROJECT_ROOT) not in sys.path:
    sys.path.insert(0, str(PROJECT_ROOT))

import django
from django.test import Client

os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'DigiGameStats.settings')

django.setup()

c = Client()
for url in ['/', '/admin/']:
    response = c.get(url)
    print(url, response.status_code)
    if response.status_code >= 400:
        print('Response content snippet:', response.content[:500])
