package config

import env "github.com/caarlos0/env/v11"

// ScannerConfig defines the config parameters for the ClusterIQ Scanner
type ScannerConfig struct {
	CloudCredentialsConfig
	APIURL string `env:"CIQ_API_URL,required"`
}

// LoadScannerConfig evaluates and return the ScannerConfig object
func LoadScannerConfig() (*ScannerConfig, error) {
	cfg := &ScannerConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
