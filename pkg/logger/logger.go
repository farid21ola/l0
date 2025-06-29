package logger

import "go.uber.org/zap"

type Config struct {
	Level string `env:"ENV"`
}

func SetupLogger(cfg Config) {
	var cfgZap zap.Config
	switch cfg.Level {
	case "local":
		cfgZap = zap.NewDevelopmentConfig()
	case "dev":
		cfgZap = zap.NewDevelopmentConfig()
	case "prod":
		cfgZap = zap.NewProductionConfig()
	default:
		cfgZap = zap.NewDevelopmentConfig()
	}

	cfgZap.DisableStacktrace = true
	logger, err := cfgZap.Build()
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(logger)
}
