package example

import (
	"log/slog"
)

func exampleSlog() {
	password := "secret123"
	apiKey := "key-abc"
	token := "tok-xyz"

	// Rule 1: uppercase start
	slog.Info("Starting server on port 8080")   // want `log message should start with a lowercase letter`
	slog.Error("Failed to connect to database") // want `log message should start with a lowercase letter`

	// Rule 1: correct
	slog.Info("starting server on port 8080")
	slog.Error("failed to connect to database")

	// Rule 2: non-English
	slog.Info("\u0437\u0430\u043f\u0443\u0441\u043a" +
		" \u0441\u0435\u0440\u0432\u0435\u0440\u0430") // want `log message should be in English only`
	slog.Error("\u043e\u0448\u0438\u0431\u043a\u0430" +
		" \u043f\u043e\u0434\u043a\u043b\u044e\u0447\u0435\u043d\u0438\u044f \u043a" +
		" \u0431\u0430\u0437\u0435 \u0434\u0430\u043d\u043d\u044b\u0445") // want `log message should be in English only`

	// Rule 2: correct
	slog.Info("starting server")
	slog.Error("failed to connect to database")

	// Rule 3: special chars/emojis
	slog.Info("server started! \U0001f680") // want `log message should not contain special characters or emojis`

	// Rule 3: correct
	slog.Info("server started")

	// Rule 4: sensitive data
	slog.Info("user password: " + password) // want `log message may contain sensitive data`
	slog.Debug("api_key=" + apiKey)         // want `log message may contain sensitive data`
	slog.Info("token: " + token)            // want `log message may contain sensitive data`

	// Rule 4: correct
	slog.Info("user authenticated successfully")
	slog.Debug("api request completed")
	slog.Info("token validated")
}
