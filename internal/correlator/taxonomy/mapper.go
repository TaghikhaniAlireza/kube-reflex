//internal/correlator/taxonomy/mapper.go
package taxonomy

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
	"gopkg.in/yaml.v3"
)

// Mapper maps raw Falco tags to a normalized MappedBehavior.
type Mapper interface {
	Map(falcoTags []string) (*model.MappedBehavior, error)
}

// MapperImpl is the concrete implementation of Mapper.
type MapperImpl struct {
	// behaviorIndex maps a MITRE ID (Txxxx or Txxxx.yyy) to its parent Tactic.
	behaviorIndex map[string]model.Tactic

	mitreIDRegex *regexp.Regexp
	noiseTags    map[string]struct{}
}

// NewMapper loads the taxonomy and prepares a strict lookup index.
func NewMapper(configPath string) (Mapper, error) {
	m := &MapperImpl{
		mitreIDRegex: regexp.MustCompile(`^T\d{4}(\.\d{3})?$`),
		noiseTags: map[string]struct{}{
			"maturity_stable":  {},
			"rule_category":    {},
			"NIST_800-53_AC-2": {},
			"PCI_DSS_10.2.3":   {},
		},
	}

	index, err := m.loadAndIndexTaxonomy(configPath)
	if err != nil {
		return nil, err
	}

	m.behaviorIndex = index
	return m, nil
}

// loadAndIndexTaxonomy flattens the nested taxonomy into an O(1) lookup table.
func (m *MapperImpl) loadAndIndexTaxonomy(path string) (map[string]model.Tactic, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read taxonomy file failed: %w", err)
	}

	var tax model.Taxonomy
	if err := yaml.Unmarshal(data, &tax); err != nil {
		return nil, fmt.Errorf("unmarshal taxonomy failed: %w", err)
	}

	index := make(map[string]model.Tactic)

	for _, tactic := range tax.Behaviors {
		for _, tech := range tactic.Techniques {

			// Technique-level (Txxxx)
			index[tech.ID] = tactic

			// Sub-technique-level (Txxxx.yyy) – highest specificity
			for _, st := range tech.SubTechniques {
				if st.ID != "" {
					index[st.ID] = tactic
				}
			}
		}
	}

	return index, nil
}

// Map resolves Falco tags into a MappedBehavior using strict MITRE priority.
func (m *MapperImpl) Map(falcoTags []string) (*model.MappedBehavior, error) {
	var selectedID string
	var contextTags []string

	for _, tag := range falcoTags {
		switch {
		case m.mitreIDRegex.MatchString(tag):
			if selectedID == "" ||
				strings.Count(tag, ".") > strings.Count(selectedID, ".") {
				selectedID = tag
			}

		case strings.HasPrefix(tag, "mitre_"):
			// Semantic MITRE tags are ignored (covered by explicit IDs)
			continue

		default:
			if _, noisy := m.noiseTags[tag]; !noisy {
				contextTags = append(contextTags, tag)
			}
		}
	}

	if selectedID == "" {
		return nil, fmt.Errorf("no valid MITRE ID found in tags: %v", falcoTags)
	}

	tactic, ok := m.behaviorIndex[selectedID]
	if !ok {
		return nil, fmt.Errorf(
			"behavior ID %s found but not defined in taxonomy",
			selectedID,
		)
	}

	return &model.MappedBehavior{
		BehaviorID:     selectedID,
		TacticID:       tactic.MitreID,
		TacticName:     tactic.Name,
		SeverityWeight: tactic.SeverityWeight,
		ContextTags:    contextTags,
	}, nil
}