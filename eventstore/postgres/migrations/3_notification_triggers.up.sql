CREATE OR REPLACE FUNCTION notify_all()
RETURNS TRIGGER
AS $$
BEGIN
    PERFORM pg_notify('$all', ''
        || '{'
        || '"stream_id": "'       || NEW.stream_id              || '" ,'
        || '"stream_type": "'     || NEW.stream_type            || '" ,'
        || '"event_type": "'      || NEW.event_type             || '" ,'
        || '"sequence_number": "' || NEW.global_sequence_number || '" ,'
        || '"version": '          || NEW."version"              || ', '
        || '"event": '            || NEW."event"::TEXT          || ', '
        || '"metadata": '         || NEW.metadata
        || '}');

    RETURN NEW;
END;
$$ LANGUAGE PLPGSQL;

CREATE OR REPLACE FUNCTION notify_stream_type()
RETURNS TRIGGER
AS $$
BEGIN
    PERFORM pg_notify(NEW.stream_type, ''
        || '{'
        || '"stream_id": "'       || NEW.stream_id              || '" ,'
        || '"stream_type": "'     || NEW.stream_type            || '" ,'
        || '"event_type": "'      || NEW.event_type             || '" ,'
        || '"sequence_number": "' || NEW.global_sequence_number || '" ,'
        || '"version": '          || NEW."version"              || ', '
        || '"event": '            || NEW."event"::TEXT          || ', '
        || '"metadata": '         || NEW.metadata
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
