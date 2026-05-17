package app

// ExecutorRegistry provides a flat lookup from connector key to executor and
// optional capability metadata.  It wraps the existing two-level
// runtimeExecutorProvider and adds connector-key-keyed helpers for the
// capability-declaration use-case.
//
// The registry is NOT a replacement for ExecutorProvider / runtimeExecutorProvider —
// those continue to handle job execution.  The registry adds:
//   - ListRegistered()   — enumerate all registered connector keys
//   - ListCapabilities() — enumerate ConnectorCapabilities per connector key
//   - GetCapabilities()  — point-lookup for a single connector key
type ExecutorRegistry struct {
	executors map[string]ChannelSyncExecutor
}

// NewExecutorRegistry returns an empty registry.
func NewExecutorRegistry() *ExecutorRegistry {
	return &ExecutorRegistry{
		executors: make(map[string]ChannelSyncExecutor),
	}
}

// Register adds an executor under its connector key.
// If the executor implements CapableExecutor the key is derived from
// ConnectorKey(); otherwise callers must provide the key explicitly via
// RegisterWithKey.
func (r *ExecutorRegistry) Register(executor CapableExecutor) {
	r.executors[executor.ConnectorKey()] = executor
}

// RegisterWithKey adds a plain ChannelSyncExecutor under an explicit key.
// Use this for executors that do not implement CapableExecutor.
func (r *ExecutorRegistry) RegisterWithKey(key string, executor ChannelSyncExecutor) {
	r.executors[key] = executor
}

// ListRegistered returns every registered connector key.
func (r *ExecutorRegistry) ListRegistered() []string {
	keys := make([]string, 0, len(r.executors))
	for k := range r.executors {
		keys = append(keys, k)
	}
	return keys
}

// ListCapabilities returns capability metadata for every registered executor
// that implements CapableExecutor.  Executors without capability declarations
// are omitted.
func (r *ExecutorRegistry) ListCapabilities() map[string]ConnectorCapabilities {
	caps := make(map[string]ConnectorCapabilities, len(r.executors))
	for k, exec := range r.executors {
		if ce, ok := exec.(CapableExecutor); ok {
			caps[k] = ce.Capabilities()
		}
	}
	return caps
}

// GetCapabilities returns the ConnectorCapabilities for a single connector key.
// Returns (ConnectorCapabilities{}, false) when the key is unknown or the
// executor does not implement CapableExecutor.
func (r *ExecutorRegistry) GetCapabilities(connectorKey string) (ConnectorCapabilities, bool) {
	exec, ok := r.executors[connectorKey]
	if !ok {
		return ConnectorCapabilities{}, false
	}
	ce, ok := exec.(CapableExecutor)
	if !ok {
		return ConnectorCapabilities{}, false
	}
	return ce.Capabilities(), true
}
