-- Backfill goal_daily_stats from existing task_goals
-- Run: surreal sql --endpoint http://localhost:8000 --username root --password root --namespace lucid --database logs < db/scripts/backfill_daily_stats.surql

DELETE goal_daily_stats;

LET $aggregated = (SELECT out AS goal_id, out.created_by AS created_by, time::floor(IF in.completed_at IS NOT NONE THEN in.completed_at ELSE in.start_date END, 1d) AS date, math::sum(quantity_value) AS daily_value, count() AS contribution_count, out.target.value AS target_value FROM task_goals WHERE out.deleted_at IS NONE GROUP BY goal_id, date);

FOR $row IN $aggregated { UPSERT goal_daily_stats SET goal_id = $row.goal_id, date = $row.date, created_by = $row.created_by, daily_value = $row.daily_value ?? 0, contribution_count = $row.contribution_count ?? 0, target_value = $row.target_value, status = IF $row.target_value IS NOT NONE AND ($row.daily_value ?? 0) >= $row.target_value THEN "met" ELSE "pending" END, updated_at = time::now() WHERE goal_id = $row.goal_id AND date = $row.date; };

SELECT count() AS total_records FROM goal_daily_stats GROUP ALL;
