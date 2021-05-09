package suites

import (
	"context"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

func TestLoadBalancer(t *testing.T) {
	feat := features.New("External Name").
		Assess("cname is reachable", func(ctx context.Context, t *testing.T) context.Context {
			return ctx
		}).Feature()
	testenv.Test(ctx, t, feat)
}
