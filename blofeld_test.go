package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestPatchSerialization(t *testing.T) {
	p := &Patch{
		Name: "Test Patch",
	}

	p.RandomizeOscillators()

	bytes, err := p.ToSDATA()
	if err != nil {
		t.Fatalf("failed to serialize patch: %v", err)
	}

	p2, err := ParseSDATA(bytes)

	if err != nil {
		t.Fatalf("failed to parse SDATA: %v", err)
	}

	if p2.Name != p.Name {
		t.Errorf("expected patch name %q, got %q", p.Name, p2.Name)
	}

	p1Json, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("failed to marshal patch to JSON: %v", err)
	}

	p2Json, err := json.Marshal(p2)
	if err != nil {
		t.Fatalf("failed to marshal parsed patch to JSON: %v", err)

	}

	if string(p1Json) == string(p2Json) {
		t.Logf("Patch JSONs are equal")
	} else {
		t.Logf("Patch JSONs are not equal")
	}

	p1Map := make(map[string]interface{})
	if err := json.Unmarshal(p1Json, &p1Map); err != nil {
		t.Fatalf("failed to unmarshal original patch JSON: %v", err)
	}

	p2Map := make(map[string]interface{})
	if err := json.Unmarshal(p2Json, &p2Map); err != nil {
		t.Fatalf("failed to unmarshal parsed patch JSON: %v", err)
	}

	keys := make(map[string]bool)
	for k := range p1Map {
		keys[k] = true
	}
	for k := range p2Map {
		keys[k] = true
	}

	for k := range keys {
		fmt.Printf("Comparing key: %s\n", k)
		if _, ok := p1Map[k]; !ok {
			t.Errorf("key %q not found in original patch JSON", k)
			continue
		}
		if _, ok := p2Map[k]; !ok {
			t.Errorf("key %q not found in parsed patch JSON", k)
			continue
		}

		if p1Map[k] == nil && p2Map[k] == nil {
			continue // Both are nil, no mismatch
		}

		type1 := fmt.Sprintf("%T", p1Map[k])
		type2 := fmt.Sprintf("%T", p2Map[k])
		if type1 != type2 {
			t.Errorf("type mismatch for key %q: expected %s, got %s", k, type1, type2)
			continue
		}

		if t, ok := p1Map[k].([]interface{}); ok {
			fmt.Printf("Comparing slice for key %q with length %d\n", k, len(t))
			continue
		}

		if p1Map[k] != p2Map[k] {
			t.Errorf("mismatch for key %q: expected %v, got %v", k, p1Map[k], p2Map[k])
		}
	}

}
