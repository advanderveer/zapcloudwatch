package main

import (
	"context"
	"log"

	"github.com/advanderveer/zapcloudwatch"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("eu-west-1"),
		config.WithSharedConfigProfile("cl-dev"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	buf := zapcloudwatch.NewCustomBuffer(10, 1048576, 86400000, 7199000, 1209999000)
	// syncer := zapcloudwatch.NewSyncer(
	// 	zapcloudwatch.Config{
	// 		AutoCreateStream: true,
	// 		LogGroupName:     "remove-me-back-test",
	// 		LogStreamName:    "remove-me-2"},
	// 	cloudwatchlogs.NewFromConfig(cfg), buf)

	// if err := syncer.Start(ctx); err != nil {
	// 	log.Fatalf("failed to start syncer: %v", err)
	// }

	cwcl := cloudwatchlogs.NewFromConfig(cfg)

	streams, _ := cwcl.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String("remove-me-back-test"),
		LogStreamNamePrefix: aws.String("remove-me"),
	})
	seqtoken := streams.LogStreams[0].UploadSequenceToken

	bw := zapcloudwatch.NewBufferedWriter(cwcl, "remove-me-back-test", "remove-me", buf, seqtoken)
	ecfg := zap.NewProductionEncoderConfig()
	enc := zapcore.NewJSONEncoder(ecfg)
	zc := zapcloudwatch.NewCore(enc, bw, zap.DebugLevel)
	logs := zap.New(zc)
	for i := 0; i < 11; i++ {
		logs.Info("foo", zap.Int("i", i))
	}

	if err := logs.Sync(); err != nil {
		log.Fatalf("failed to sync: %v", err)
	}

	// if err := syncer.Stop(ctx); err != nil {
	// 	log.Fatalf("failed to stop: %v", err)
	// }

}
