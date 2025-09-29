package handlers

import (
	"testing"
)

func TestParsePhoneList(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"5527997799027@c.us", []string{"5527997799027@c.us"}},
		{"5527997799027@c.us,5527997799028@c.us", []string{"5527997799027@c.us", "5527997799028@c.us"}},
		{" 5527997799027@c.us , 5527997799028@c.us ", []string{"5527997799027@c.us", "5527997799028@c.us"}},
		{"5527997799027@c.us,,5527997799028@c.us", []string{"5527997799027@c.us", "5527997799028@c.us"}},
	}

	for _, test := range tests {
		result := parsePhoneList(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("parsePhoneList(%q) returned %d items, expected %d", test.input, len(result), len(test.expected))
			continue
		}
		for i, phone := range result {
			if phone != test.expected[i] {
				t.Errorf("parsePhoneList(%q)[%d] = %q, expected %q", test.input, i, phone, test.expected[i])
			}
		}
	}
}

func TestFindPhoneDifference(t *testing.T) {
	tests := []struct {
		list1    []string
		list2    []string
		expected []string
	}{
		{[]string{}, []string{}, []string{}},
		{[]string{"A"}, []string{}, []string{"A"}},
		{[]string{}, []string{"A"}, []string{}},
		{[]string{"A", "B"}, []string{"A"}, []string{"B"}},
		{[]string{"A", "B", "C"}, []string{"B", "D"}, []string{"A", "C"}},
	}

	for _, test := range tests {
		result := findPhoneDifference(test.list1, test.list2)
		if len(result) != len(test.expected) {
			t.Errorf("findPhoneDifference(%v, %v) returned %d items, expected %d", test.list1, test.list2, len(result), len(test.expected))
			continue
		}

		// Convert to map for easier comparison
		resultMap := make(map[string]bool)
		for _, phone := range result {
			resultMap[phone] = true
		}

		for _, expectedPhone := range test.expected {
			if !resultMap[expectedPhone] {
				t.Errorf("findPhoneDifference(%v, %v) missing expected phone %q", test.list1, test.list2, expectedPhone)
			}
		}
	}
}
