package config

import "github.com/joho/godotenv"

// LoadDotEnv loads optional dotenv files without replacing variables already set by the process.
func LoadDotEnv(path string) error { return godotenv.Load(path) }
