module github.com/kds-rebalance

go 1.23

require (
	github.com/aws/aws-sdk-go v1.41.7
	github.com/aws/aws-sdk-go-v2 v1.39.6
	github.com/aws/aws-sdk-go-v2/config v1.31.20
	github.com/aws/aws-sdk-go-v2/credentials v1.18.24
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.42.3
	github.com/vmware/vmware-go-kcl v1.5.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.3 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.40.2 // indirect
	github.com/aws/smithy-go v1.23.2 // indirect
	github.com/awslabs/kinesis-aggregation/go v0.0.0-20210630091500-54e17340d32f // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	golang.org/x/sys v0.1.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)

replace github.com/vmware/vmware-go-kcl => github.com/ns-nagaaravindb/vmware-go-kcl v1.5.1
