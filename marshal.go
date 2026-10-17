package i18n

func (m I18n) MarshalJSON() ([]byte, error) {
	switch len(m) {
	case 0:
		return []byte(`""`), nil
	case 1:
		for _, v := range m {
			return Marshal(v)
		}
	}
	return Marshal(map[string]string(m))
}

func (m *I18n) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*m = nil
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := Unmarshal(b, &s); err != nil {
			return err
		}
		*m = I18n{"und": s} // und = undetermined
		return nil
	}
	var mm map[string]string
	if err := Unmarshal(b, &mm); err != nil {
		return err
	}
	*m = mm
	return nil
}
