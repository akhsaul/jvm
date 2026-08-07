package env

import (
	"fmt"
	"os"
)

// Set mengatur environment variable untuk proses saat ini.
// Catatan: perubahan ini hanya berlaku untuk proses jwrapper itu sendiri.
// Untuk mengatur env di shell yang memanggil, user perlu men-source
// script yang dihasilkan oleh perintah 'use'.
func Set(key, value string) error {
	if err := os.Setenv(key, value); err != nil {
		return fmt.Errorf("gagal set env %s: %w", key, err)
	}
	fmt.Printf("Set %s=%s\n", key, value)
	return nil
}

// Get mengambil nilai environment variable.
func Get(key string) string {
	return os.Getenv(key)
}
