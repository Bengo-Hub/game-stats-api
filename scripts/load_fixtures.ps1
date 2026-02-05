$files = @(
  "scripts/fixtures/core_world.json",
  "scripts/fixtures/core_continent.json",
  "scripts/fixtures/core_country.json",
  "scripts/fixtures/core_location.json",
  "scripts/fixtures/core_field.json",
  "scripts/fixtures/events_discipline.json",
  "scripts/fixtures/events_event.json",
  "scripts/fixtures/events_divisionpool.json",
  "scripts/fixtures/games_gameround.json",
  "scripts/fixtures/games_team.json",
  "scripts/fixtures/games_player.json",
  "scripts/fixtures/games_game_fixed.json",
  "scripts/fixtures/games_scoring.json",
  "scripts/fixtures/games_spiritscore.json",
  "scripts/fixtures/auth_permission.json",
  "scripts/fixtures/auth_group.json",
  "scripts/fixtures/authman_user.json"
)

foreach ($f in $files) {
  Write-Host "Loading $f"
  python manage.py loaddata $f
}

Write-Host "All fixtures loaded."