package tests

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	"go.uber.org/zap/zapcore"

	"github.com/k8sbykeshed/k8s-service-validator/matrix"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const dnsDomain = "cluster.local"

var (
	// flags
	debug     bool
	namespace string

	manager *matrix.KubeManager
	testenv env.Environment

	model *matrix.Model
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Enable debug log level.")
	flag.StringVar(&namespace, "namespace", matrix.GetNamespace(), "Set namespace used to run the tests.")
}

// NewLoggerConfig return the configuration object for the logger
func NewLoggerConfig(options ...zap.Option) *zap.Logger {
	logLevel := zap.InfoLevel
	if debug {
		logLevel = zap.DebugLevel
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		NameKey:     "logger",
		TimeKey:     "timer",
		EncodeLevel: zapcore.CapitalColorLevelEncoder,
		EncodeTime:  zapcore.RFC3339TimeEncoder,
	}), os.Stdout, logLevel)
	return zap.New(core).WithOptions(options...)
}

func TestMain(m *testing.M) {
	cfg, err := envconf.NewFromFlags()
	if err != nil {
		log.Fatalf("envconf failed: %s", err)
	}

	zap.ReplaceGlobals(NewLoggerConfig())

	clientSet, config := matrix.NewClientSet()
	manager = matrix.NewKubeManager(clientSet, config)
	namespace = matrix.GetNamespace()

	// Setup brings the pods only in the start, all tests share the same pods
	// access them via different services types.
	testenv = env.NewWithConfig(cfg)
	testenv.Setup(
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			var (
				nodes []*v1.Node
				pods  []string
			)

			if nodes, err = manager.GetReadyNodes(); err != nil {
				log.Fatal(err)
			}

			// Generate pod names using existing nodes
			for i := 1; i <= len(nodes); i++ {
				pods = append(pods, fmt.Sprintf("pod-%d", i))
			}

			// Initialize environment pods model and cluster.
			model = matrix.NewModel([]string{namespace}, pods, []int32{80, 81}, []v1.Protocol{v1.ProtocolTCP, v1.ProtocolUDP}, dnsDomain)
			if err = manager.StartPods(model, nodes); err != nil {
				log.Fatal(err)
			}

			// Wait until HTTP servers are up.
			if err = manager.WaitForHTTPServers(model); err != nil {
				log.Fatal(err)
			}
			return ctx, nil
		},
	).Finish(
		// Finished cleans up the namespace in the end of the suite.
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			zap.L().Info("Cleanup namespace.", zap.String("namespace", namespace))
			if err := manager.DeleteNamespaces([]string{namespace}); err != nil {
				log.Fatal(err)
			}
			return ctx, nil
		},
	)

	os.Exit(testenv.Run(m))
}
