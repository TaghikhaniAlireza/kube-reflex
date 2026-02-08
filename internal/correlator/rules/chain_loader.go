// internal/correlator/rules/chain_loader.go
package rules

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

type ChainLoader struct {
	path string
}

func NewChainLoader(path string) *ChainLoader {
	return &ChainLoader{path: path}
}

// Load reads chains.yml, parses it and validates it
func (l *ChainLoader) Load() (*model.ChainFile, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read chains file: %w", err)
	}

	var file model.ChainFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse chains yaml: %w", err)
	}

	if err := validateChainFile(&file); err != nil {
		return nil, err
	}

	return &file, nil
}

func validateChainFile(file *model.ChainFile) error {
	if file.Version <= 0 {
		return fmt.Errorf("invalid chains file version")
	}

	if len(file.Chains) == 0 {
		return fmt.Errorf("no chains defined")
	}

	seenIDs := make(map[string]struct{})

	for _, chain := range file.Chains {
		if chain.ID == "" {
			return fmt.Errorf("chain with empty id detected")
		}

		if _, exists := seenIDs[chain.ID]; exists {
			return fmt.Errorf("duplicate chain id detected: %s", chain.ID)
		}
		seenIDs[chain.ID] = struct{}{}

		if len(chain.Sequence) < 2 {
			return fmt.Errorf("chain %s must have at least 2 tactics", chain.ID)
		}

		if chain.Severity == "" {
			return fmt.Errorf("chain %s has empty severity", chain.ID)
		}

		if chain.MaxDuration == "" {
			return fmt.Errorf("chain %s has empty max_duration", chain.ID)
		}
	}

	return nil
}