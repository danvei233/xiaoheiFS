.PHONY: run docs proto demo-plugins

run:
	go run ./cmd/server

docs:
	go run ./cmd/tools/gendocs

proto:
	protoc -I . \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		plugin/v1/types.proto \
		plugin/v1/manifest.proto \
		plugin/v1/core.proto \
		plugin/v1/sms.proto \
		plugin/v1/kyc.proto \
		plugin/v1/payment.proto \
		plugin/v1/automation.proto

demo-plugins:
	go build -o plugins/payment/ezpay/plugin.exe ./plugin-demo/pluginv1/payment_ezpay
	go build -o plugins/sms/alisms/plugin.exe ./plugin-demo/pluginv1/sms_alisms_mock
	go build -o plugins/sms/tencent_sms/plugin.exe ./plugin-demo/pluginv1/sms_tencent_mock
	go build -o plugins/kyc/aliyun_kyc/plugin.exe ./plugin-demo/pluginv1/kyc_aliyun_mock
	go build -o plugins/kyc/tencent_kyc/plugin.exe ./plugin-demo/pluginv1/kyc_tencent_mock
	go build -o plugins/automation/xiaohei_proxy/plugin.exe ./plugin-demo/pluginv1/automation_xiaohei_proxy
	go build -o plugins/automation/mofang_openapi/plugin.exe ./plugin-demo/pluginv1/automation_mofang_openapi
