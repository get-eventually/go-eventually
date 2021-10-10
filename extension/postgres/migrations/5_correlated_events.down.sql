DROP TRIGGER           IF EXISTS project_correlated_event_on_append ON events;
DROP FUNCTION          IF EXISTS project_correlated_event;
DROP MATERIALIZED VIEW IF EXISTS correlated_events_view;
DROP TABLE             IF EXISTS correlated_events;
