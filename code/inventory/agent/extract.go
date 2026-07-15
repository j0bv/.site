package agent

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// Criteria holds extracted search criteria. Main converts this to SearchCriteria.
type Criteria struct {
	YearMin   *int
	YearMax   *int
	Make      *string
	Model     *string
	MaxPrice  *float64
	MinPrice  *float64
	MaxMiles  *int
	Location  *string
}

type criteriaJSON struct {
	YearMin  *int     `json:"year_min"`
	YearMax  *int     `json:"year_max"`
	Make     *string  `json:"make"`
	Model    *string  `json:"model"`
	MaxPrice *float64 `json:"max_price"`
	MinPrice *float64 `json:"min_price"`
	MaxMiles *int     `json:"max_miles"`
	Location *string  `json:"location"`
}

// ExtractCriteria calls the SLM when SLM_ENABLED is true, parses JSON into Criteria,
// normalizes make/model to uppercase, and omits missing fields. Returns nil on error or empty.
func ExtractCriteria(ctx context.Context, subject, body string) (*Criteria, error) {
	if os.Getenv("SLM_ENABLED") == "false" {
		return nil, nil
	}

	prompt := "You are extracting truck search criteria from a buyer's email. Reply with a single JSON object only. Allowed keys: year_min, year_max, make, model, max_price, min_price, max_miles, location. Use integers for years and miles, floats for price, strings for make/model/location. Omit any key not clearly stated. Schema aligns with truck_listings: year, make, model, price, miles, location.\n\nExample: {\"make\":\"PETERBILT\",\"max_price\":100000}\n\nEmail subject: " + subject + "\n\nEmail body: " + body
	if subject == "" && body == "" {
		prompt = "Reply with only: {}"
	}

	content, err := Chat(ctx, prompt)
	if err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, nil
	}

	// Slurp JSON from optional markdown code block
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return nil, nil
	}

	var cj criteriaJSON
	if err := json.Unmarshal([]byte(jsonStr), &cj); err != nil {
		return nil, err
	}

	c := &Criteria{
		YearMin:  cj.YearMin,
		YearMax:  cj.YearMax,
		MaxPrice: cj.MaxPrice,
		MinPrice: cj.MinPrice,
		MaxMiles: cj.MaxMiles,
		Location: cj.Location,
	}
	if cj.Make != nil {
		s := strings.ToUpper(*cj.Make)
		c.Make = &s
	}
	if cj.Model != nil {
		s := strings.ToUpper(*cj.Model)
		c.Model = &s
	}
	return c, nil
}

var jsonBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*([\\s\\S]*?)```")

func extractJSON(s string) string {
	if m := jsonBlockRegex.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	// Try whole string as JSON
	s = strings.TrimSpace(s)
	if (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) || (strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) {
		return s
	}
	// Find first { to last }
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	end := strings.LastIndex(s, "}")
	if end < start {
		return ""
	}
	return s[start : end+1]
}
