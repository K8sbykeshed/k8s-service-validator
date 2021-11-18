package suites

import (
	"context"
	"fmt"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"testing"

	"github.com/k8sbykeshed/k8s-service-lb-validator/matrix"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/e2e-framework/pkg/env"
)

const DNS_DOMAIN = "cluster.local"

var (
	namespace string
	config    *rest.Config
	testenv   env.Environment
	model     *matrix.Model
	cs        *kubernetes.Clientset
	ma        *matrix.KubeManager
	ctx       = context.Background()
	logger    *zap.Logger
)

func init() {
	logger = NewLoggerConfig()
}

func NewLoggerConfig(options ...zap.Option) *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		TimeKey:        "timer",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	// todo(knabben) - flag to enable debugging level
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), os.Stdout, zap.InfoLevel)
	return zap.New(core).WithOptions(options...)
}

// TestMain sets the general before/after function hooks
func TestMain(m *testing.M) {
	var (
		err   error
		nodes []*v1.Node
	)

	cs, config = matrix.NewClientSet()
	cfg, err := envconf.NewFromFlags()
	if err != nil {
		log.Fatalf("envconf failed: %s", err)
	}

	testenv = env.NewWithConfig(cfg)
	ma = matrix.NewKubeManager(cs, config, logger)
	namespace = matrix.GetNamespace()

	// Setup brings the pods only in the start, all tests share the same pods
	// access them via different services types.
	testenv.Setup(
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			var pods []string
			if nodes, err = ma.GetReadyNodes(); err != nil {
				log.Fatal(err)
			}

			// Generate pod names using existing nodes
			for i := 1; i <= len(nodes); i++ {
				pods = append(pods, fmt.Sprintf("pod-%d", i))
			}

			// Initialize environment pods model and cluster.
			model = matrix.NewModel([]string{namespace}, pods, []int32{80, 81}, []v1.Protocol{v1.ProtocolTCP, v1.ProtocolUDP}, DNS_DOMAIN)
			if err = ma.StartPods(model, nodes); err != nil {
				log.Fatal(err)
			}

			// Wait until HTTP servers are up.
			if err = ma.WaitForHTTPServers(model); err != nil {
				log.Fatal(err)
			}
			return ctx, nil
		},
	).Finish(
		// Finished cleans up the namespace in the end of the suite.
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			logger.Info("Cleanup namespace.", zap.String("namespace", namespace))
			if err := ma.DeleteNamespaces([]string{namespace}); err != nil {
				log.Fatal(err)
			}
			return ctx, nil
		},
	)

	os.Exit(testenv.Run(m))
}
