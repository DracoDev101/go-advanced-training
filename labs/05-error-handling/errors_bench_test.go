package errlab

import (
	"database/sql"
	"errors"
	"testing"
)

var boolSink bool
var bodySink ErrorBody

func BenchmarkErrorsIsWrapped(b *testing.B) {
	err := WrapWithPercentW(WrapWithPercentW(sql.ErrNoRows))
	for i := 0; i < b.N; i++ {
		boolSink = errors.Is(err, sql.ErrNoRows)
	}
}

func BenchmarkAPIErrorFor(b *testing.B) {
	err := WrapWithPercentW(ErrUserNotFound)
	for i := 0; i < b.N; i++ {
		bodySink = APIErrorFor(err, "req")
	}
}
