package gosmig

import (
	"context"
	"fmt"
	"io"
	"time"
)

func runCmdVersion[TDBRow DBRow, TDBResult DBResult, TDBOrTX DBOrTX[TDBRow, TDBResult]](
	ctx context.Context,
	dbOrTX TDBOrTX,
	output io.Writer,
	timeout time.Duration,
) error {

	dbVersion, err := getDBVersion(ctx, dbOrTX, timeout)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(output, "Current database version:\n%d\n", dbVersion)
	return nil
}
