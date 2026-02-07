package taxonomy

import (
	"reflect"
	"testing"
	"strings"
)

// NOTE: Test file path must be relative to the running test binary, 
// which is usually the package directory.
const testConfigPath = "./behaviors.yml"

func TestMapperInitialization(t *testing.T) {
	_, err := NewMapper(testConfigPath)
	if err != nil {
		t.Fatalf("NewMapper failed to initialize: %v", err)
	}
}

// TestMapperPriority checks if the most specific MITRE ID (sub-technique) is selected.
func TestMapper_SubTechniquePriority(t *testing.T) {
	mapper, err := NewMapper(testConfigPath)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// T1595.002 is more specific than T1595.
	tags := []string{
		"T1595", 
		"T1595.002",
		"mitre_reconnaissance",
		"container", // Non-MITRE Tag
		"shell",     // Non-MITRE Tag
	}

	result, err := mapper.Map(tags)
	if err != nil {
		t.Fatalf("Map failed: %v", err)
	}

	// 1. Verify Highest Priority ID
	expectedBehaviorID := "T1595.002"
	if result.BehaviorID != expectedBehaviorID {
		t.Errorf("Priority selection error: Expected %s, Got %s", expectedBehaviorID, result.BehaviorID)
	}

	// 2. Verify Tactic and Weight are correctly inherited from the Tactic struct (TA0043)
	if result.TacticID != "TA0043" {
		t.Errorf("TacticID inheritance error: Expected TA0043, Got %s", result.TacticID)
	}
	if result.SeverityWeight != 3 {
		t.Errorf("SeverityWeight inheritance error: Expected 3, Got %d", result.SeverityWeight)
	}

	// 3. Verify ContextTags are extracted correctly (noise/general mitre tags filtered)
	expectedContextTags := []string{"container", "shell"}
	if !reflect.DeepEqual(result.ContextTags, expectedContextTags) {
		t.Errorf("ContextTags error: Expected %v, Got %v", expectedContextTags, result.ContextTags)
	}
}

// TestMapper_MitreNotFound checks correct error handling when no MITRE tag is present.
func TestMapper_MitreNotFound(t *testing.T) {
	mapper, err := NewMapper(testConfigPath)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	tags := []string{
		"container", 
		"rule_category", // Noise tag
	}

	_, err = mapper.Map(tags)
	if err == nil || !strings.Contains(err.Error(), "no valid MITRE ID found in tags: [container rule_category]") {
		t.Errorf("Expected 'no valid MITRE behavior ID found' error, got: %v", err)
	}
}