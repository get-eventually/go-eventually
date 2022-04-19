package postgres

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/jackc/pgconn"

	"github.com/get-eventually/go-eventually/core/version"
)

var versionConflictErrorRegex = regexp.MustCompile(
	`(aggregate|event stream) version check failed, expected: (?P<expected>\d), got: (?P<got>\d)`,
)

func isVersionConflictError(err error) (version.ConflictError, bool) {
	var pgErr *pgconn.PgError

	if err == nil || !errors.As(err, &pgErr) {
		return version.ConflictError{}, false
	}

	matches := versionConflictErrorRegex.FindStringSubmatch(pgErr.Message)
	if len(matches) == 0 {
		return version.ConflictError{}, false
	}

	expected, err := strconv.Atoi(matches[versionConflictErrorRegex.SubexpIndex("expected")])
	if err != nil {
		return version.ConflictError{}, false
	}

	got, err := strconv.Atoi(matches[versionConflictErrorRegex.SubexpIndex("got")])
	if err != nil {
		return version.ConflictError{}, false
	}

	return version.ConflictError{
		Expected: version.Version(expected),
		Actual:   version.Version(got),
	}, true
}
