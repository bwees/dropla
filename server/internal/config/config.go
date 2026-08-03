package config

import "os"

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Port() string {
	return env("PORT", "3000")
}

func DataDir() string {
	return env("DATA_DIR", ".")
}

func WebStaticPath() string {
	return os.Getenv("WEB_STATIC_PATH")
}
