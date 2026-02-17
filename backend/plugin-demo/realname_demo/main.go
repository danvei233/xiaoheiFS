package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"xiaoheiplay/internal/adapter/realname"
	"xiaoheiplay/internal/app/shared"
)

type DemoProvider struct{}

func (p *DemoProvider) Key() string {
	return "demo_realname"
}

func (p *DemoProvider) Name() string {
	return "Demo Realname Provider"
}

func (p *DemoProvider) Verify(ctx context.Context, realName string, idNumber string) (bool, string, error) {
	realName = strings.TrimSpace(realName)
	idNumber = strings.TrimSpace(idNumber)
	if realName == "" || idNumber == "" {
		return false, "real name and id number required", nil
	}
	if len(idNumber) < 6 {
		return false, "id number too short", nil
	}
	return true, "", nil
}

func main() {
	reg := realname.NewRegistry()
	reg.Register(&DemoProvider{})

	provider, err := reg.GetProvider("demo_realname")
	if err != nil {
		fmt.Println("provider not found:", err)
		os.Exit(1)
	}

	ok, reason, err := provider.Verify(context.Background(), "Alice", "ABC123456")
	if err != nil {
		fmt.Println("verify error:", err)
		os.Exit(1)
	}
	fmt.Printf("verified=%v reason=%q\n", ok, reason)

	_ = shared.RealNameProvider(provider)
}
