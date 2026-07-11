package observe_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/goloop/observe"
)

// ExampleRegistry wires health, readiness and build endpoints onto a mux.
func ExampleRegistry() {
	obs := observe.New(
		observe.WithBuildInfo(observe.BuildInfo{Service: "api", Version: "v1.2.3"}),
		observe.WithErrorRedaction(true),
	)

	// Register dependency checks (database, cache, ...).
	obs.Check("cache", func(context.Context) error { return nil })

	mux := http.NewServeMux()
	mux.Handle("/healthz", obs.HealthHandler())
	mux.Handle("/readyz", obs.ReadyHandler())
	mux.Handle("/buildz", obs.BuildHandler())

	st := obs.Ready(context.Background())
	fmt.Println(st.Status, st.Service, st.Version)
	// Output: ok api v1.2.3
}

// ExampleRegistry_lifecycleBridge shows registering an external lifecycle
// snapshot as an ordinary check, so observe stays decoupled from the kernel
// that produced it.
func ExampleRegistry_lifecycleBridge() {
	obs := observe.New()

	// healthyFromKernel would come from your lifecycle kernel's status.
	healthyFromKernel := func() bool { return true }

	obs.Check("app", func(context.Context) error {
		if !healthyFromKernel() {
			return errors.New("a component failed")
		}
		return nil
	})

	fmt.Println(obs.Ready(context.Background()).OK())
	// Output: true
}
