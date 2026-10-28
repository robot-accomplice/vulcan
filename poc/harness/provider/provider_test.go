package provider

import "testing"

func TestRead(t *testing.T) {
	if b := Read("https://example.test/anthropic", "x", 3); b.Error == "" || b.Order != 3 {
		t.Fatalf("%+v", b)
	}
	b, _, err := endpoints["api.deepseek.com"].read([]byte(`{"is_available":true,"balance_infos":[{"currency":"USD","total_balance":"17.91"}]}`))
	if err != nil || b != 17.91 {
		t.Fatalf("%v %v", b, err)
	}
	o, _, err := endpoints["openrouter.ai"].read([]byte(`{"data":{"total_credits":20,"total_usage":1.5}}`))
	if err != nil || o != 18.5 {
		t.Fatalf("%v %v", o, err)
	}
}
