package main

import "context"

type TfConfigProvider interface {
	DownloadTfConfig(ctx context.Context, versionId *string, path *string) error
}
