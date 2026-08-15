package analyzers

import (
	"fmt"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]DocumentAnalyzer)
)

func Register(analyzer DocumentAnalyzer) {
	registryMu.Lock()
	defer registryMu.Unlock()
	info := analyzer.Info()
	registry[info.ID] = analyzer
}

func Get(id string) (DocumentAnalyzer, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	a, ok := registry[id]
	return a, ok
}

func List() []AnalyzerInfo {
	registryMu.RLock()
	defer registryMu.RUnlock()
	result := make([]AnalyzerInfo, 0, len(registry))
	for _, a := range registry {
		result = append(result, a.Info())
	}
	return result
}

func GetAnalyzer(id string) (DocumentAnalyzer, error) {
	a, ok := Get(id)
	if !ok {
		return nil, fmt.Errorf("analyzer %q not found", id)
	}
	return a, nil
}
