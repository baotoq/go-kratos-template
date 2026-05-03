package data

import (
	"github.com/google/wire"
)

// ProviderSet is data providers — only infra primitives, no biz imports.
var ProviderSet = wire.NewSet(NewWorkflowClient)
