-- Modify "division_pools" table
ALTER TABLE "division_pools" DROP CONSTRAINT "division_pools_events_division_pools";
-- Modify "teams" table
ALTER TABLE "teams" DROP CONSTRAINT "teams_division_pools_teams";
-- Create "division_pool_teams" table
CREATE TABLE "division_pool_teams" ("division_pool_id" uuid NOT NULL, "team_id" uuid NOT NULL, PRIMARY KEY ("division_pool_id", "team_id"), CONSTRAINT "division_pool_teams_division_pool_id" FOREIGN KEY ("division_pool_id") REFERENCES "division_pools" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "division_pool_teams_team_id" FOREIGN KEY ("team_id") REFERENCES "teams" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create "event_division_pools" table
CREATE TABLE "event_division_pools" ("event_id" uuid NOT NULL, "division_pool_id" uuid NOT NULL, PRIMARY KEY ("event_id", "division_pool_id"), CONSTRAINT "event_division_pools_division_pool_id" FOREIGN KEY ("division_pool_id") REFERENCES "division_pools" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "event_division_pools_event_id" FOREIGN KEY ("event_id") REFERENCES "events" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Modify "game_rounds" table
ALTER TABLE "game_rounds" DROP CONSTRAINT "game_rounds_events_game_rounds";
-- Create "event_game_rounds" table
CREATE TABLE "event_game_rounds" ("event_id" uuid NOT NULL, "game_round_id" uuid NOT NULL, PRIMARY KEY ("event_id", "game_round_id"), CONSTRAINT "event_game_rounds_event_id" FOREIGN KEY ("event_id") REFERENCES "events" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "event_game_rounds_game_round_id" FOREIGN KEY ("game_round_id") REFERENCES "game_rounds" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Modify "games" table
ALTER TABLE "games" ADD COLUMN "event_games" uuid NULL, ADD CONSTRAINT "games_events_games" FOREIGN KEY ("event_games") REFERENCES "events" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "scorings" table
ALTER TABLE "scorings" ADD COLUMN "team_id" uuid NULL, ADD CONSTRAINT "scorings_teams_team" FOREIGN KEY ("team_id") REFERENCES "teams" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "players" table
ALTER TABLE "players" DROP CONSTRAINT "players_teams_players";
-- Create "team_players" table
CREATE TABLE "team_players" ("team_id" uuid NOT NULL, "player_id" uuid NOT NULL, PRIMARY KEY ("team_id", "player_id"), CONSTRAINT "team_players_player_id" FOREIGN KEY ("player_id") REFERENCES "players" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "team_players_team_id" FOREIGN KEY ("team_id") REFERENCES "teams" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
