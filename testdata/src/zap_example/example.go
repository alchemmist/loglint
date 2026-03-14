package zap_example

import "go.uber.org/zap"

func exampleZap(logger *zap.Logger, sugar *zap.SugaredLogger) {
	token := "tok-xyz"

	// Rule 1: uppercase start
	logger.Info("Starting server")        // want `log message should start with a lowercase letter`
	sugar.Infow("Connection established") // want `log message should start with a lowercase letter`

	// Rule 1: correct
	logger.Info("starting server")
	sugar.Infow("connection established")

	// Rule 3: special chars
	sugar.Infof("done! \U0001f680") // want `log message should not contain special characters or emojis`

	// Rule 4: sensitive data
	sugar.Infof("token: " + token) // want `log message may contain sensitive data`

	// Correct
	logger.Info("server started successfully")
	sugar.Infow("request processed", "status", 200)
}
