package plugins

import "testing"

// TestStaticSchemaContractPins pins the static config schemas to the field set
// returned by each plugin process (backend/plugin-demo/pluginv1). Regression
// for the schema drift: static files missing fields or carrying stale required
// lists / defaults.
func TestStaticSchemaContractPins(t *testing.T) {
	t.Run("ezpay exposes order_expire_minutes", func(t *testing.T) {
		s := loadSchema(t, "payment/ezpay/schemas/config.schema.json")
		m := prop(t, s, "order_expire_minutes")
		if num(t, m, "default") != 5 {
			t.Errorf("order_expire_minutes default = %v, want 5", m["default"])
		}
	})

	t.Run("alisms exposes region/endpoint/timeout_sec", func(t *testing.T) {
		s := loadSchema(t, "sms/alisms/schemas/config.schema.json")
		for _, field := range []string{"region", "endpoint", "timeout_sec"} {
			if _, ok := s.Properties[field]; !ok {
				t.Errorf("alisms schema missing property %q", field)
			}
		}
	})

	t.Run("tencent_sms exposes region/timeout_sec", func(t *testing.T) {
		s := loadSchema(t, "sms/tencent_sms/schemas/config.schema.json")
		for _, field := range []string{"region", "timeout_sec"} {
			if _, ok := s.Properties[field]; !ok {
				t.Errorf("tencent_sms schema missing property %q", field)
			}
		}
	})

	t.Run("openidc_default exposes default_nic_type and timeout range", func(t *testing.T) {
		s := loadSchema(t, "automation/openidc_default/schemas/config.schema.json")
		m := prop(t, s, "default_nic_type")
		if got := strSlice(t, m, "enum"); !setEqual(got, []string{"nat", "pub", "none"}) {
			t.Errorf("default_nic_type enum = %v, want [nat pub none]", got)
		}
		to := prop(t, s, "timeout_sec")
		if num(t, to, "default") != 60 || num(t, to, "maximum") != 600 {
			t.Errorf("timeout_sec default/max = %v/%v, want 60/600", to["default"], to["maximum"])
		}
	})

	t.Run("xiaohei_proxy prefers open_key over deprecated open_secret", func(t *testing.T) {
		s := loadSchema(t, "automation/xiaohei_proxy/schemas/config.schema.json")
		if _, ok := s.Properties["open_key"]; !ok {
			t.Error("xiaohei_proxy schema missing property open_key")
		}
		for _, field := range s.Required {
			if field == "open_secret" {
				t.Error("xiaohei_proxy schema must not require the deprecated open_secret")
			}
		}
		if !setEqual(s.Required, []string{"base_url", "open_akid", "goods_type_id"}) {
			t.Errorf("xiaohei_proxy required = %v, want [base_url open_akid goods_type_id]", s.Required)
		}
	})
}
