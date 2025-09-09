package indexer

import (
	"encoding/json"
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"
)

type rbacAnnotationProcessor struct {}

func (p *rbacAnnotationProcessor) isJSON(annotationValue string) bool {
	var m json.RawMessage
	b := []byte(annotationValue)
	return json.Unmarshal(b, &m) == nil
}

func (p *rbacAnnotationProcessor) processAsJSON(annotationValue string) []string {
	claimsMap := make(map[string]any)
	json.Unmarshal([]byte(annotationValue), &claimsMap)
	return p.processMap(claimsMap)
}

func (p *rbacAnnotationProcessor) isMultiLineString(annotationValue string) bool {
	return strings.Contains(annotationValue, "\n")
}

func (p *rbacAnnotationProcessor) processAsMultiLineString(annotationValue string) []string {
	refinedClaimValues := []string{}
	for e := range strings.SplitSeq(annotationValue, "\n") {
		if e != "" {
			clean := func(s string) string {
				s = strings.TrimSpace(s)              // rm spaces
				return strings.ReplaceAll(s, "'", "") // rm single quotes
			}
			lastColonIndex := strings.LastIndex(e, ":")
			if lastColonIndex != -1 { // protect from panicing on invalid input
				claimKey := clean(e[:lastColonIndex])
				claimValues := strings.SplitSeq(e[lastColonIndex+1:], ",")
				for cv := range claimValues {
					if claimValue := clean(cv); claimValue != "" {
						refinedClaimValues = append(refinedClaimValues, FormatClaim(claimKey, claimValue))
					}
				}
			}
		}
	}
	return refinedClaimValues
}

func (p *rbacAnnotationProcessor) processAsYAML(annotationValue string) []string {
	claimsMap := make(map[string]any)
	yaml.Unmarshal([]byte(annotationValue), &claimsMap)
	return p.processMap(claimsMap)
}

func (p *rbacAnnotationProcessor) processMap(m map[string]any) []string {
	var refinedClaimValues []string
	for k, v := range m {
		switch val := v.(type) {
		case string:
			if strings.Contains(val, ",") {
				for v := range strings.SplitSeq(val, ",") {
					refinedClaimValues = append(refinedClaimValues,
						FormatClaim(k, v),
					)
				}
				continue
			}
			refinedClaimValues = append(refinedClaimValues,
				FormatClaim(k, strings.TrimSpace(val)),
			)
		case []any:
			for _, val := range val {
				refinedClaimValues = append(refinedClaimValues,
					FormatClaim(k, strings.TrimSpace(fmt.Sprintf("%v", val))),
				)
			}
		}
	}
	return refinedClaimValues
}
