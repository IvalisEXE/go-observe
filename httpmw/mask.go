package httpmw

import "encoding/json"

// maskJSON parse body sebagai JSON dan ganti value field sensitif jadi "***".
// Kalau body bukan JSON valid (misal form-data/binary), balikin placeholder
// biar ga nge-log data mentah yang ga jelas / berat.
func maskJSON(body []byte, sensitiveFields []string) []byte {
	if len(body) == 0 {
		return body
	}

	var parsed interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return []byte(`"<non-json-body skipped>"`)
	}

	sensSet := make(map[string]bool, len(sensitiveFields))
	for _, f := range sensitiveFields {
		sensSet[f] = true
	}

	masked := maskValue(parsed, sensSet)
	out, _ := json.Marshal(masked)
	return out
}

func maskValue(v interface{}, sens map[string]bool) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		for k, sub := range val {
			if sens[k] {
				val[k] = "***"
				continue
			}
			val[k] = maskValue(sub, sens)
		}
		return val
	case []interface{}:
		for i, sub := range val {
			val[i] = maskValue(sub, sens)
		}
		return val
	default:
		return val
	}
}
