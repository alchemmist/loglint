package config_yaml

import "log/slog"

func Example() {
	token := "tok"

	slog.Info("server started!") // want `log message should not contain special characters or emojis`

	slog.Info("Starting server") // no lowercase check when config is loaded
	slog.Info("запуск сервера")  // no english check when config is loaded
	slog.Info("token: " + token) // no sensitive check when config is loaded
}
