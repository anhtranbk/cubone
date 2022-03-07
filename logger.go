package cubone

import "go.uber.org/zap"

var logger, _ = zap.NewDevelopment()
var log = logger.Sugar()
