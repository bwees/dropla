//go:build !unix

package storage

func freeBytes(string) (int64, error) { return 0, nil }
