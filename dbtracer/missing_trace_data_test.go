package dbtracer

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Each End hook reads its trace data from the context the matching Start hook
// returned. Reached with a context that never went through Start, the lookup
// has to return instead of panicking.
func TestEndHooksWithoutStartData(t *testing.T) {
	tracer, err := NewDBTracer("test")
	if err != nil {
		t.Fatalf("NewDBTracer: %v", err)
	}

	ctx := context.Background()
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
	tracer.TraceBatchQuery(ctx, nil, pgx.TraceBatchQueryData{})
	tracer.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{})
	tracer.TraceConnectEnd(ctx, pgx.TraceConnectEndData{})
	tracer.TraceCopyFromEnd(ctx, nil, pgx.TraceCopyFromEndData{})
	tracer.TracePrepareEnd(ctx, nil, pgx.TracePrepareEndData{})
	tracer.TraceAcquireEnd(ctx, nil, pgxpool.TraceAcquireEndData{})
}
