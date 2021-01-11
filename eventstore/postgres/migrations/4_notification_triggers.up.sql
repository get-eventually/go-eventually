CREATE OR REPLACE FUNCTION notify_all()
RETURNS TRIGGER
AS $$
DECLARE
    metadata JSONB;
BEGIN
    metadata = NEW.metadata ||
        ('{"Global-Sequence-Number":' || NEW.global_sequence_number || ' }')::JSONB;

    PERFORM pg_notify('$all', ''
        || '{'
        || '"stream_id": "'      || NEW.stream_id           || '" ,'
        || '"stream_type": "'    || NEW.stream_type         || '" ,'
        || '"event_type": "'     || NEW.event_type          || '" ,'
        || '"version": '         || NEW."version"           || ', '
        || '"event": '           || NEW."event"::TEXT       || ', '
        || '"metadata": '        || metadata
        || '}');

    RETURN NEW;
END;
$$ LANGUAGE PLPGSQL;

CREATE OR REPLACE FUNCTION notify_stream_type()
RETURNS TRIGGER
AS $$
DECLARE
    metadata JSONB;
BEGIN
    metadata = NEW.metadata ||
        ('{"Global-Sequence-Number":' || NEW.global_sequence_number || ' }')::JSONB;

    PERFORM pg_notify(NEW.stream_type, ''
        || '{'
        || '"stream_id": "'      || NEW.stream_id           || '" ,'
        || '"stream_type": "'    || NEW.stream_type         || '" ,'
        || '"event_type": "'     || NEW.event_type          || '" ,'
        || '"version": '         || NEW."version"           || ', '
        || '"event": '           || NEW."event"::TEXT       || ', '
        || '"metadata": '        || metadata
        || '}');

    RETURN NEW;
END;
$$ LANGUAGE PLPGSQL;

CREATE TRIGGER notify_all_on_append
    AFTER INSERT
    ON events
    FOR EACH ROW
    EXECUTE PROCEDURE notify_all();

CREATE TRIGGER notify_stream_type_on_append
    AFTER INSERT
    ON events
    FOR EACH ROW
    EXECUTE PROCEDURE notify_stream_type();
