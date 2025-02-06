package config

// SecretStringValue must be exported - used in tests.
const SecretStringValue = "<secret>"

// SecretString is a type that should be used for fields that should not be visible in logs.
type SecretString string

// MarshalJSON marshals SecretString to JSON making sure that actual value is not visible.
func (s SecretString) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return []byte("\"" + SecretStringValue + "\""), nil
}

// MarshalYAML marshals SecretString to YAML making sure that actual value is not visible.
func (s SecretString) MarshalYAML() (interface{}, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return SecretStringValue, nil
}
