package suites

import (
	"context"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"testing"
)

func TestClusterIP(t *testing.T) {
	feat := features.New("Cluster IP").
		Assess("pods are reachable", func(ctx context.Context, t *testing.T) context.Context {
			return ctx
		}).Feature()
	testenv.Test(ctx, t, feat)
}
