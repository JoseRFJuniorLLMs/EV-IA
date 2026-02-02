package config

import "time"

type Config struct {
	App            AppConfig            `mapstructure:"app"`
	HTTP           HTTPConfig           `mapstructure:"http"`
	GRPC           GRPCConfig           `mapstructure:"grpc"`
	OCPP           OCPPConfig           `mapstructure:"ocpp"`
	Database       DatabaseConfig       `mapstructure:"database"`
	Redis          RedisConfig          `mapstructure:"redis"`
	NATS           NATSConfig           `mapstructure:"nats"`
	JWT            JWTConfig            `mapstructure:"jwt"`
	Gemini         GeminiConfig         `mapstructure:"gemini"`
	OpenTelemetry  OpenTelemetryConfig  `mapstructure:"opentelemetry"`
	Prometheus     PrometheusConfig     `mapstructure:"prometheus"`
	Logging        LoggingConfig        `mapstructure:"logging"`
	RateLimiting   RateLimitingConfig   `mapstructure:"rate_limiting"`
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
	CORS           CORSConfig           `mapstructure:"cors"`
	Security       SecurityConfig       `mapstructure:"security"`
	Payment        PaymentConfig        `mapstructure:"payment"`
	Notification   NotificationConfig   `mapstructure:"notification"`
	Analytics      AnalyticsConfig      `mapstructure:"analytics"`
	FeatureFlags   FeatureFlagsConfig   `mapstructure:"feature_flags"`
	Cache          CacheConfig          `mapstructure:"cache"`
	Jobs           JobsConfig           `mapstructure:"jobs"`
	Limits         LimitsConfig         `mapstructure:"limits"`
	Region         RegionConfig         `mapstructure:"region"`
	Compliance     ComplianceConfig     `mapstructure:"compliance"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
}

type HTTPConfig struct {
	Port           int           `mapstructure:"port"`
	AllowedOrigins []string      `mapstructure:"allowed_origins"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout"`
}

type GRPCConfig struct {
	Port           int `mapstructure:"port"`
	MaxConnections int `mapstructure:"max_connections"`
}

type OCPPConfig struct {
	Port                  int           `mapstructure:"port"`
	Version               string        `mapstructure:"version"`
	HeartbeatInterval     int           `mapstructure:"heartbeat_interval"`
	WebsocketPingInterval time.Duration `mapstructure:"websocket_ping_interval"`
	Security              OCPPSecurity  `mapstructure:"security"`
}

type OCPPSecurity struct {
	Enabled    bool   `mapstructure:"enabled"`
	TLSCert    string `mapstructure:"tls_cert"`
	TLSKey     string `mapstructure:"tls_key"`
	ClientAuth bool   `mapstructure:"client_auth"`
}

type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	AutoMigrate     bool          `mapstructure:"auto_migrate"`
	LogQueries      bool          `mapstructure:"log_queries"`
}

type RedisConfig struct {
	URL          string        `mapstructure:"url"`
	MaxRetries   int           `mapstructure:"max_retries"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	PoolTimeout  time.Duration `mapstructure:"pool_timeout"`
}

type NATSConfig struct {
	URL           string        `mapstructure:"url"`
	MaxReconnects int           `mapstructure:"max_reconnects"`
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
	Timeout       time.Duration `mapstructure:"timeout"`
}

type JWTConfig struct {
	Secret               string        `mapstructure:"secret"`
	AccessTokenDuration  time.Duration `mapstructure:"access_token_duration"`
	RefreshTokenDuration time.Duration `mapstructure:"refresh_token_duration"`
	Issuer               string        `mapstructure:"issuer"`
	Audience             string        `mapstructure:"audience"`
}

type GeminiConfig struct {
	APIKey            string            `mapstructure:"api_key"`
	Model             string            `mapstructure:"model"`
	VoiceConfig       GeminiVoiceConfig `mapstructure:"voice_config"`
	SystemInstruction string            `mapstructure:"system_instruction"`
}

type GeminiVoiceConfig struct {
	VoiceName        string `mapstructure:"voice_name"`
	Language         string `mapstructure:"language"`
	SpeechModel      string `mapstructure:"speech_model"`
	ResponseModality string `mapstructure:"response_modality"`
}

type OpenTelemetryConfig struct {
	Enabled     bool              `mapstructure:"enabled"`
	Jaeger      JaegerConfig      `mapstructure:"jaeger"`
	ServiceName string            `mapstructure:"service_name"`
	Attributes  map[string]string `mapstructure:"attributes"`
}

type JaegerConfig struct {
	Endpoint     string  `mapstructure:"endpoint"`
	SamplerType  string  `mapstructure:"sampler_type"`
	SamplerParam float64 `mapstructure:"sampler_param"`
}

type PrometheusConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
	Port    int    `mapstructure:"port"`
}

type LoggingConfig struct {
	Level    string          `mapstructure:"level"`
	Format   string          `mapstructure:"format"`
	Output   string          `mapstructure:"output"`
	Sampling LoggingSampling `mapstructure:"sampling"`
}

type LoggingSampling struct {
	Enabled    bool `mapstructure:"enabled"`
	Initial    int  `mapstructure:"initial"`
	Thereafter int  `mapstructure:"thereafter"`
}

type RateLimitingConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	MaxRequests int           `mapstructure:"max_requests"`
	Window      time.Duration `mapstructure:"window"`
	ByUser      bool          `mapstructure:"by_user"`
}

type CircuitBreakerConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	MaxRequests      int           `mapstructure:"max_requests"`
	Interval         time.Duration `mapstructure:"interval"`
	Timeout          time.Duration `mapstructure:"timeout"`
	FailureThreshold float64       `mapstructure:"failure_threshold"`
}

type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	ExposeHeaders  []string `mapstructure:"expose_headers"`
	MaxAge         int      `mapstructure:"max_age"`
	Credentials    bool     `mapstructure:"credentials"`
}

type SecurityConfig struct {
	EnableHTTPS bool   `mapstructure:"enable_https"`
	TLSCertPath string `mapstructure:"tls_cert_path"`
	TLSKeyPath  string `mapstructure:"tls_key_path"`
	EnableMTLS  bool   `mapstructure:"enable_mtls"`
	CACertPath  string `mapstructure:"ca_cert_path"`
}

type PaymentConfig struct {
	Stripe  StripeConfig  `mapstructure:"stripe"`
	Pricing PricingConfig `mapstructure:"pricing"`
}

type StripeConfig struct {
	SecretKey     string `mapstructure:"secret_key"`
	WebhookSecret string `mapstructure:"webhook_secret"`
	Currency      string `mapstructure:"currency"`
}

type PricingConfig struct {
	PerKWh           float64 `mapstructure:"per_kwh"`
	IdleFeePerMinute float64 `mapstructure:"idle_fee_per_minute"`
}

type NotificationConfig struct {
	Email EmailConfig `mapstructure:"email"`
	SMS   SMSConfig   `mapstructure:"sms"`
	Push  PushConfig  `mapstructure:"push"`
}

type EmailConfig struct {
	Provider string `mapstructure:"provider"`
	APIKey   string `mapstructure:"api_key"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
}

type SMSConfig struct {
	Provider   string `mapstructure:"provider"`
	AccountSID string `mapstructure:"account_sid"`
	AuthToken  string `mapstructure:"auth_token"`
	From       string `mapstructure:"from"`
}

type PushConfig struct {
	Provider        string `mapstructure:"provider"`
	CredentialsPath string `mapstructure:"credentials_path"`
}

type AnalyticsConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	BatchSize     int           `mapstructure:"batch_size"`
	FlushInterval time.Duration `mapstructure:"flush_interval"`
	Providers     []string      `mapstructure:"providers"`
}

type FeatureFlagsConfig struct {
	VoiceAssistant  bool `mapstructure:"voice_assistant"`
	SmartCharging   bool `mapstructure:"smart_charging"`
	V2G             bool `mapstructure:"v2g"`
	BlockchainAudit bool `mapstructure:"blockchain_audit"`
}

type CacheConfig struct {
	DeviceStatusTTL       time.Duration `mapstructure:"device_status_ttl"`
	UserSessionTTL        time.Duration `mapstructure:"user_session_ttl"`
	TransactionSummaryTTL time.Duration `mapstructure:"transaction_summary_ttl"`
	AnalyticsTTL          time.Duration `mapstructure:"analytics_ttl"`
}

type JobsConfig struct {
	DailyReport          JobSchedule `mapstructure:"daily_report"`
	AnalyticsAggregation JobSchedule `mapstructure:"analytics_aggregation"`
	DeviceHealthCheck    JobSchedule `mapstructure:"device_health_check"`
	InvoiceGeneration    JobSchedule `mapstructure:"invoice_generation"`
}

type JobSchedule struct {
	Schedule string `mapstructure:"schedule"`
	Enabled  bool   `mapstructure:"enabled"`
}

type LimitsConfig struct {
	MaxActiveSessionsPerUser int           `mapstructure:"max_active_sessions_per_user"`
	MaxTransactionDuration   time.Duration `mapstructure:"max_transaction_duration"`
	MaxFileUploadSize        string        `mapstructure:"max_file_upload_size"`
	MaxRequestBodySize       string        `mapstructure:"max_request_body_size"`
}

type RegionConfig struct {
	Timezone string `mapstructure:"timezone"`
	Locale   string `mapstructure:"locale"`
	Currency string `mapstructure:"currency"`
}

type ComplianceConfig struct {
	GDPREnabled       bool `mapstructure:"gdpr_enabled"`
	DataRetentionDays int  `mapstructure:"data_retention_days"`
	AuditLogEnabled   bool `mapstructure:"audit_log_enabled"`
	PIIEncryption     bool `mapstructure:"pii_encryption"`
}
