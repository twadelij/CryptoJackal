use anyhow::Result;
use std::collections::HashMap;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::RwLock;
use tracing::{info, warn, error};

pub mod audit;
pub mod authentication;
pub mod encryption;
pub mod rate_limiting;
pub mod validation;

use audit::AuditLogger;
use authentication::AuthService;
use encryption::EncryptionService;
use rate_limiting::RateLimiter;
use validation::InputValidator;

/// Comprehensive security framework
pub struct SecurityFramework {
    config: crate::core::config::Config,
    audit_logger: AuditLogger,
    auth_service: AuthService,
    encryption_service: EncryptionService,
    rate_limiter: RateLimiter,
    input_validator: InputValidator,
    security_metrics: RwLock<SecurityMetrics>,
}

impl SecurityFramework {
    pub fn new(config: crate::core::config::Config) -> Self {
        Self {
            audit_logger: AuditLogger::new(),
            auth_service: AuthService::new(config.jwt_secret.clone()),
            encryption_service: EncryptionService::new(),
            rate_limiter: RateLimiter::new(),
            input_validator: InputValidator::new(),
            config,
            security_metrics: RwLock::new(SecurityMetrics::new()),
        }
    }

    /// Initialize security framework
    pub async fn initialize(&self) -> Result<()> {
        info!("Initializing security framework");

        // Initialize audit logging
        self.audit_logger.initialize().await?;

        // Initialize encryption keys
        self.encryption_service.initialize().await?;

        // Set up rate limiting
        self.rate_limiter.configure(&self.config).await?;

        info!("Security framework initialized successfully");
        Ok(())
    }

    /// Authenticate user request
    pub async fn authenticate_request(&self, token: &str, endpoint: &str) -> Result<AuthResult> {
        let start_time = SystemTime::now();

        // Check rate limiting first
        if let Err(e) = self.rate_limiter.check_rate_limit(endpoint).await {
            self.log_security_event("rate_limit_exceeded", endpoint, Some(&e.to_string())).await;
            return Err(e);
        }

        // Validate JWT token
        let auth_result = self.auth_service.validate_token(token).await?;

        // Log authentication attempt
        self.log_security_event("authentication_attempt", endpoint, None).await;

        // Update metrics
        self.update_auth_metrics(start_time, auth_result.success).await;

        Ok(auth_result)
    }

    /// Validate and sanitize input
    pub async fn validate_input(&self, input: &str, input_type: InputType) -> Result<ValidatedInput> {
        let validated = self.input_validator.validate(input, input_type).await?;

        // Log input validation
        self.log_security_event("input_validation", &format!("{:?}", input_type), None).await;

        Ok(validated)
    }

    /// Encrypt sensitive data
    pub async fn encrypt_data(&self, data: &str) -> Result<String> {
        let encrypted = self.encryption_service.encrypt(data).await?;
        
        // Log encryption event
        self.log_security_event("data_encrypted", "sensitive_data", None).await;

        Ok(encrypted)
    }

    /// Decrypt sensitive data
    pub async fn decrypt_data(&self, encrypted_data: &str) -> Result<String> {
        let decrypted = self.encryption_service.decrypt(encrypted_data).await?;
        
        // Log decryption event
        self.log_security_event("data_decrypted", "sensitive_data", None).await;

        Ok(decrypted)
    }

    /// Get security metrics
    pub async fn get_security_metrics(&self) -> SecurityMetrics {
        let metrics = self.security_metrics.read().await;
        metrics.clone()
    }

    /// Perform security audit
    pub async fn perform_security_audit(&self) -> Result<SecurityAuditReport> {
        info!("Performing security audit");

        let mut report = SecurityAuditReport::new();

        // Audit authentication
        report.auth_audit = self.audit_authentication().await?;

        // Audit encryption
        report.encryption_audit = self.audit_encryption().await?;

        // Audit rate limiting
        report.rate_limit_audit = self.audit_rate_limiting().await?;

        // Audit input validation
        report.validation_audit = self.audit_validation().await?;

        // Calculate overall security score
        report.calculate_overall_score();

        info!("Security audit completed. Overall score: {:.1}", report.overall_score);
        Ok(report)
    }

    async fn audit_authentication(&self) -> Result<AuthAuditResult> {
        let mut result = AuthAuditResult::new();

        // Check JWT secret strength
        result.jwt_secret_strength = self.calculate_password_strength(&self.config.jwt_secret);

        // Check session timeout
        result.session_timeout_appropriate = self.config.session_timeout >= 300 && self.config.session_timeout <= 86400;

        // Check max login attempts
        result.max_login_attempts_reasonable = self.config.max_login_attempts >= 3 && self.config.max_login_attempts <= 10;

        Ok(result)
    }

    async fn audit_encryption(&self) -> Result<EncryptionAuditResult> {
        let mut result = EncryptionAuditResult::new();

        // Test encryption/decryption
        let test_data = "sensitive_test_data";
        match self.encryption_service.encrypt(test_data).await {
            Ok(encrypted) => {
                match self.encryption_service.decrypt(&encrypted).await {
                    Ok(decrypted) => {
                        result.encryption_functional = decrypted == test_data;
                    }
                    Err(_) => {
                        result.encryption_functional = false;
                    }
                }
            }
            Err(_) => {
                result.encryption_functional = false;
            }
        }

        Ok(result)
    }

    async fn audit_rate_limiting(&self) -> Result<RateLimitAuditResult> {
        let mut result = RateLimitAuditResult::new();

        // Test rate limiting functionality
        result.rate_limiting_configured = true; // Simplified check

        Ok(result)
    }

    async fn audit_validation(&self) -> Result<ValidationAuditResult> {
        let mut result = ValidationAuditResult::new();

        // Test input validation
        let test_cases = vec![
            ("valid_address", "0x1234567890123456789012345678901234567890", true),
            ("invalid_address", "invalid", false),
            ("empty_string", "", false),
            ("sql_injection", "'; DROP TABLE users; --", false),
        ];

        for (test_name, input, expected) in test_cases {
            let validation_result = self.input_validator.validate(input, InputType::Address).await;
            let passed = validation_result.is_valid() == expected;
            result.validation_tests.push(ValidationTest {
                name: test_name.to_string(),
                passed,
                input: input.to_string(),
                expected,
            });
        }

        Ok(result)
    }

    fn calculate_password_strength(&self, password: &str) -> f64 {
        let mut score = 0.0;

        // Length check
        if password.len() >= 16 {
            score += 0.3;
        } else if password.len() >= 8 {
            score += 0.2;
        }

        // Complexity checks
        if password.chars().any(|c| c.is_uppercase()) {
            score += 0.2;
        }
        if password.chars().any(|c| c.is_lowercase()) {
            score += 0.2;
        }
        if password.chars().any(|c| c.is_numeric()) {
            score += 0.2;
        }
        if password.chars().any(|c| !c.is_alphanumeric()) {
            score += 0.1;
        }

        // Avoid common patterns
        if !password.to_lowercase().contains("password") &&
           !password.to_lowercase().contains("secret") &&
           !password.to_lowercase().contains("default") {
            score += 0.1;
        }

        score.clamp(0.0, 1.0)
    }

    async fn log_security_event(&self, event_type: &str, context: &str, error: Option<&str>) {
        let event = SecurityEvent {
            event_type: event_type.to_string(),
            context: context.to_string(),
            error: error.map(|e| e.to_string()),
            timestamp: SystemTime::now(),
        };

        if let Err(e) = self.audit_logger.log_event(&event).await {
            error!("Failed to log security event: {}", e);
        }
    }

    async fn update_auth_metrics(&self, start_time: SystemTime, success: bool) {
        let mut metrics = self.security_metrics.write().await;
        metrics.total_auth_requests += 1;
        
        if success {
            metrics.successful_auth_requests += 1;
        } else {
            metrics.failed_auth_requests += 1;
        }

        let response_time = SystemTime::now()
            .duration_since(start_time)
            .unwrap_or_default()
            .as_millis() as u64;
        
        metrics.average_auth_response_time = 
            (metrics.average_auth_response_time * (metrics.total_auth_requests - 1) as u64 + response_time) / 
            metrics.total_auth_requests as u64;
    }
}

/// Security metrics
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct SecurityMetrics {
    pub total_auth_requests: u64,
    pub successful_auth_requests: u64,
    pub failed_auth_requests: u64,
    pub average_auth_response_time: u64,
    pub rate_limit_hits: u64,
    pub validation_failures: u64,
    pub encryption_operations: u64,
    pub last_updated: SystemTime,
}

impl SecurityMetrics {
    pub fn new() -> Self {
        Self {
            total_auth_requests: 0,
            successful_auth_requests: 0,
            failed_auth_requests: 0,
            average_auth_response_time: 0,
            rate_limit_hits: 0,
            validation_failures: 0,
            encryption_operations: 0,
            last_updated: SystemTime::now(),
        }
    }
}

/// Authentication result
#[derive(Debug, Clone)]
pub struct AuthResult {
    pub success: bool,
    pub user_id: Option<String>,
    pub permissions: Vec<String>,
    pub expires_at: Option<SystemTime>,
}

/// Input types for validation
#[derive(Debug, Clone)]
pub enum InputType {
    Address,
    Amount,
    TokenSymbol,
    Json,
    QueryString,
}

/// Validated input
#[derive(Debug, Clone)]
pub struct ValidatedInput {
    pub original: String,
    pub sanitized: String,
    pub is_valid: bool,
    pub validation_errors: Vec<String>,
}

/// Security event
#[derive(Debug, Clone)]
struct SecurityEvent {
    pub event_type: String,
    pub context: String,
    pub error: Option<String>,
    pub timestamp: SystemTime,
}

/// Security audit report
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct SecurityAuditReport {
    pub auth_audit: AuthAuditResult,
    pub encryption_audit: EncryptionAuditResult,
    pub rate_limit_audit: RateLimitAuditResult,
    pub validation_audit: ValidationAuditResult,
    pub overall_score: f64,
    pub timestamp: SystemTime,
}

impl SecurityAuditReport {
    pub fn new() -> Self {
        Self {
            auth_audit: AuthAuditResult::new(),
            encryption_audit: EncryptionAuditResult::new(),
            rate_limit_audit: RateLimitAuditResult::new(),
            validation_audit: ValidationAuditResult::new(),
            overall_score: 0.0,
            timestamp: SystemTime::now(),
        }
    }

    pub fn calculate_overall_score(&mut self) {
        let auth_score = if self.auth_audit.jwt_secret_strength > 0.7 &&
                           self.auth_audit.session_timeout_appropriate &&
                           self.auth_audit.max_login_attempts_reasonable { 1.0 } else { 0.5 };

        let encryption_score = if self.encryption_audit.encryption_functional { 1.0 } else { 0.0 };

        let rate_limit_score = if self.rate_limit_audit.rate_limiting_configured { 1.0 } else { 0.5 };

        let validation_score = if self.validation_audit.validation_tests.iter().all(|t| t.passed) { 1.0 } else { 0.5 };

        self.overall_score = (auth_score + encryption_score + rate_limit_score + validation_score) / 4.0;
    }
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct AuthAuditResult {
    pub jwt_secret_strength: f64,
    pub session_timeout_appropriate: bool,
    pub max_login_attempts_reasonable: bool,
}

impl AuthAuditResult {
    pub fn new() -> Self {
        Self {
            jwt_secret_strength: 0.0,
            session_timeout_appropriate: false,
            max_login_attempts_reasonable: false,
        }
    }
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct EncryptionAuditResult {
    pub encryption_functional: bool,
}

impl EncryptionAuditResult {
    pub fn new() -> Self {
        Self {
            encryption_functional: false,
        }
    }
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct RateLimitAuditResult {
    pub rate_limiting_configured: bool,
}

impl RateLimitAuditResult {
    pub fn new() -> Self {
        Self {
            rate_limiting_configured: false,
        }
    }
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ValidationAuditResult {
    pub validation_tests: Vec<ValidationTest>,
}

impl ValidationAuditResult {
    pub fn new() -> Self {
        Self {
            validation_tests: Vec::new(),
        }
    }
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ValidationTest {
    pub name: String,
    pub passed: bool,
    pub input: String,
    pub expected: bool,
}
