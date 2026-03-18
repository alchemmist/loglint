package config_json

import "log/slog"

func Example() {
	token := "tok"

	slog.Info("запуск сервера") // want `log message should be in English only`

	slog.Info("Starting server!") // no lowercase/special check when config is loaded
	slog.Info("token: " + token)  // no sensitive check when config is loaded
}
