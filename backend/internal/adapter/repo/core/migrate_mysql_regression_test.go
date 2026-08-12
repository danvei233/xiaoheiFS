package repo

import (
	"reflect"
	"strings"
	"testing"
)

func TestPackageRow_IntegrationPackageID_UsesNonUniqueIndexTag(t *testing.T) {
	field, ok := reflect.TypeOf(packageRow{}).FieldByName("IntegrationPackageID")
	if !ok {
		t.Fatal("packageRow.IntegrationPackageID field not found")
	}
	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "index:idx_packages_integration") {
		t.Fatalf("expected non-unique index tag idx_packages_integration, got: %s", tag)
	}
	if strings.Contains(tag, "uniqueIndex") {
		t.Fatalf("expected no uniqueIndex on IntegrationPackageID, got: %s", tag)
	}
}

func TestUserTierAutoRuleRow_ConditionsJSON_HasNoDefaultValueTag(t *testing.T) {
	field, ok := reflect.TypeOf(userTierAutoRuleRow{}).FieldByName("ConditionsJSON")
	if !ok {
		t.Fatal("userTierAutoRuleRow.ConditionsJSON field not found")
	}
	tag := field.Tag.Get("gorm")
	if strings.Contains(tag, "default:") {
		t.Fatalf("expected no default for conditions_json, got: %s", tag)
	}
	// No explicit type pin: MySQL maps the plain string to LONGTEXT while
	// Postgres/SQLite keep TEXT. Pinning type:text here would make MySQL's
	// AutoMigrate shrink the upgraded LONGTEXT back to TEXT every startup.
	if strings.Contains(tag, "type:text") {
		t.Fatalf("expected no TEXT type pin for conditions_json, got: %s", tag)
	}
}

func TestCouponProductGroupRow_RulesJSON_HasNoDefaultValueTag(t *testing.T) {
	field, ok := reflect.TypeOf(couponProductGroupRow{}).FieldByName("RulesJSON")
	if !ok {
		t.Fatal("couponProductGroupRow.RulesJSON field not found")
	}
	tag := field.Tag.Get("gorm")
	if strings.Contains(tag, "default:") {
		t.Fatalf("expected no default for TEXT column rules_json, got: %s", tag)
	}
	if !strings.Contains(tag, "type:text") {
		t.Fatalf("expected TEXT type tag for rules_json, got: %s", tag)
	}
}
