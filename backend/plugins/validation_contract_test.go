package plugins

import "testing"

// TestValidationContractPins guards the constraints that the static schemas
// must carry to match the documented/validated contract of each plugin:
// wechatpay_v3 requires a 32-char API v3 key, ezpay only signs with MD5.
func TestValidationContractPins(t *testing.T) {
	t.Run("wechatpay_v3 api_v3_key is length-constrained", func(t *testing.T) {
		s := loadSchema(t, "payment/wechatpay_v3/schemas/config.schema.json")
		m := prop(t, s, "api_v3_key")
		if num(t, m, "minLength") != 32 || num(t, m, "maxLength") != 32 {
			t.Errorf("api_v3_key minLength/maxLength = %v/%v, want 32/32", m["minLength"], m["maxLength"])
		}
	})

	t.Run("ezpay sign_type is restricted to MD5", func(t *testing.T) {
		s := loadSchema(t, "payment/ezpay/schemas/config.schema.json")
		m := prop(t, s, "sign_type")
		if got := strSlice(t, m, "enum"); !setEqual(got, []string{"MD5"}) {
			t.Errorf("sign_type enum = %v, want [MD5]", got)
		}
	})
}
