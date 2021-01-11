DROP TRIGGER IF EXISTS notify_stream_type_on_append ON events;
DROP TRIGGER IF EXISTS notify_all_on_append ON events;

DROP FUNCTION IF EXISTS notify_stream_type;
DROP FUNCTION IF EXISTS notify_all;
