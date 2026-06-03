package backend

import "os"

func envExists(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}
