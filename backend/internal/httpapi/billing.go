package httpapi

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Select one alternative per billable dimension. Profile prices override model
// prices; input-token and output-image rules remain separate, additive charges.
func selectPriceRules(rules []priceRule, parameters response) ([]priceRule, error) {
	groups := map[string][]priceRule{}
	for _, rule := range rules {
		if rule.Price < 0 || rule.ProviderPrice < 0 || math.IsNaN(rule.Price) || math.IsInf(rule.Price, 0) || math.IsNaN(rule.ProviderPrice) || math.IsInf(rule.ProviderPrice, 0) {
			return nil, fmt.Errorf("invalid_price_rule: %s", rule.ID)
		}
		var metadata response
		if err := json.Unmarshal([]byte(rule.PriceMetadata), &metadata); err != nil {
			return nil, fmt.Errorf("invalid_price_metadata: %s", rule.ID)
		}
		matches := true
		for _, key := range []string{"resolution", "quality", "size"} {
			if expected, ok := metadata[key]; ok {
				if actual, supplied := parameters[key]; supplied && fmt.Sprint(actual) != fmt.Sprint(expected) {
					matches = false
				}
			}
		}
		if variant, ok := parameters["pricing_variant"]; ok && fmt.Sprint(variant) != rule.PricingVariant {
			matches = false
		}
		key := rule.PriceType + ":" + rule.PriceUnit
		// Keep the dimension even when none of its alternatives match.
		if _, ok := groups[key]; !ok {
			groups[key] = nil
		}
		if matches {
			groups[key] = append(groups[key], rule)
		}
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	selected := []priceRule{}
	for _, key := range keys {
		candidates := groups[key]
		hasProfile := false
		for _, rule := range candidates {
			hasProfile = hasProfile || rule.ProfileMatched
		}
		filtered := []priceRule{}
		for _, rule := range candidates {
			if !hasProfile || rule.ProfileMatched {
				filtered = append(filtered, rule)
			}
		}
		if len(filtered) != 1 {
			return nil, fmt.Errorf("pricing_variant_required_or_unavailable: %s", key)
		}
		selected = append(selected, filtered[0])
	}
	return selected, nil
}

// Reject invalid quantities instead of silently truncating or billing defaults.
func validateBillingParameters(parameters response) error {
	for _, key := range []string{"output_count", "duration_seconds", "max_tokens", "max_completion_tokens"} {
		value, ok := parameters[key]
		if !ok {
			continue
		}
		number, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 1 || number != math.Trunc(number) || number > 1e9 {
			return fmt.Errorf("invalid_billing_parameter: %s", key)
		}
		if key == "output_count" && number > 4 {
			return fmt.Errorf("output_count_must_be_between_1_and_4")
		}
	}
	return nil
}
