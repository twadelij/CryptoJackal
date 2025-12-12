use anyhow::Result;
use axum::{
    extract::State,
    http::StatusCode,
    response::Json,
    routing::{get, post},
    Router,
};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;
use tokio::process::Command;
use tracing::{info, warn, error};

use crate::api::ApiState;

/// Setup configuration structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetupConfig {
    pub environment: String,
    pub node_url: String,
    pub chain_id: String,
    pub trade_amount: String,
    pub scan_interval: String,
    pub gas_limit: String,
    pub slippage_tolerance: String,
    pub min_liquidity: String,
    pub paper_trading_mode: bool,
    pub coingecko_api_key: Option<String>,
    pub dexscreener_api_key: Option<String>,
    pub telegram_bot_token: Option<String>,
    pub discord_webhook_url: Option<String>,
    pub jwt_secret: String,
    pub cors_origins: String,
}

/// Setup response
#[derive(Debug, Serialize)]
pub struct SetupResponse {
    pub success: bool,
    pub message: String,
    pub data: Option<serde_json::Value>,
}

/// Deployment status
#[derive(Debug, Serialize)]
pub struct DeploymentStatus {
    pub status: String,
    pub message: String,
    pub progress: u8,
    pub services: Vec<ServiceStatus>,
}

#[derive(Debug, Serialize)]
pub struct ServiceStatus {
    pub name: String,
    pub status: String,
    pub url: Option<String>,
}

/// Create setup routes
pub fn create_setup_routes() -> Router<ApiState> {
    Router::new()
        .route("/validate", post(validate_config))
        .route("/save-config", post(save_config))
        .route("/deploy", post(deploy_setup))
        .route("/status", get(deployment_status))
        .route("/health", get(setup_health))
}

/// Validate configuration
pub async fn validate_config(
    State(state): State<ApiState>,
    Json(config): Json<SetupConfig>,
) -> Result<Json<SetupResponse>, StatusCode> {
    info!("Validating setup configuration");

    let mut errors = Vec::new();

    // Validate required fields
    if config.node_url.is_empty() {
        errors.push("Node URL is required");
    } else if !config.node_url.starts_with("http") {
        errors.push("Node URL must start with http:// or https://");
    }

    if config.chain_id.is_empty() {
        errors.push("Chain ID is required");
    }

    if config.trade_amount.is_empty() || config.trade_amount.parse::<u64>().is_err() {
        errors.push("Invalid trade amount");
    }

    if config.jwt_secret.is_empty() {
        errors.push("JWT secret is required");
    } else if config.jwt_secret.len() < 32 {
        errors.push("JWT secret must be at least 32 characters");
    }

    // Environment-specific validation
    if config.environment == "production" {
        if config.paper_trading_mode {
            errors.push("Paper trading should be disabled in production");
        }
        
        if config.jwt_secret == "default-secret" || config.jwt_secret.contains("default") {
            errors.push("Using default JWT secret is not allowed in production");
        }
    }

    if errors.is_empty() {
        Ok(Json(SetupResponse {
            success: true,
            message: "Configuration is valid".to_string(),
            data: None,
        }))
    } else {
        Ok(Json(SetupResponse {
            success: false,
            message: format!("Validation failed: {}", errors.join(", ")),
            data: Some(serde_json::json!({ "errors": errors })),
        }))
    }
}

/// Save configuration to environment file
pub async fn save_config(
    State(state): State<ApiState>,
    Json(config): Json<SetupConfig>,
) -> Result<Json<SetupResponse>, StatusCode> {
    info!("Saving setup configuration");

    // First validate
    let validation_result = validate_config(State(state.clone()), Json(config.clone())).await?;
    if !validation_result.success {
        return Ok(validation_result);
    }

    // Create environment file content
    let env_content = generate_env_file(&config);

    // Write to .env file
    let env_path = Path::new(".env");
    
    // Backup existing file if it exists
    if env_path.exists() {
        let backup_path = format!(".env.backup.{}", chrono::Utc::now().format("%Y%m%d_%H%M%S"));
        if let Err(e) = fs::copy(env_path, &backup_path) {
            error!("Failed to backup existing .env file: {}", e);
            return Ok(Json(SetupResponse {
                success: false,
                message: "Failed to backup existing configuration".to_string(),
                data: None,
            }));
        }
        info!("Existing .env file backed up to {}", backup_path);
    }

    // Write new configuration
    if let Err(e) = fs::write(env_path, env_content) {
        error!("Failed to write .env file: {}", e);
        return Ok(Json(SetupResponse {
            success: false,
            message: "Failed to save configuration file".to_string(),
            data: None,
        }));
    }

    info!("Configuration saved successfully");

    Ok(Json(SetupResponse {
        success: true,
        message: "Configuration saved successfully".to_string(),
        data: Some(serde_json::json!({
            "environment": config.environment,
            "paper_trading": config.paper_trading_mode,
            "chain_id": config.chain_id
        })),
    }))
}

/// Generate environment file content
fn generate_env_file(config: &SetupConfig) -> String {
    let mut content = format!(
        r#"# =============================================================================
# CryptoJackal Environment Configuration
# =============================================================================
# Environment: {}
# Generated: {}
# =============================================================================

# --------------------------------------------------------------
# Node Configuration
# --------------------------------------------------------------
NODE_URL={}
CHAIN_ID={}
NETWORK_NAME=ethereum

# --------------------------------------------------------------
# Trading Parameters
# --------------------------------------------------------------
SCAN_INTERVAL={}
GAS_LIMIT={}
SLIPPAGE_TOLERANCE={}
MIN_LIQUIDITY={}
MAX_PRICE_IMPACT=0.02
TRADE_AMOUNT={}

# --------------------------------------------------------------
# Paper Trading Configuration
# --------------------------------------------------------------
PAPER_TRADING_MODE={}
PAPER_TRADING_BALANCE=10.0
PAPER_TRADING_DATA_SOURCE=historical

# --------------------------------------------------------------
# API Configuration
# --------------------------------------------------------------
API_HOST=0.0.0.0
API_PORT=8080
API_BASE_URL=http://localhost:8080
CORS_ORIGINS={}

# --------------------------------------------------------------
# Security Configuration
# --------------------------------------------------------------
JWT_SECRET={}
SESSION_TIMEOUT=3600
MAX_LOGIN_ATTEMPTS=5

# --------------------------------------------------------------
# Logging Configuration
# --------------------------------------------------------------
LOG_LEVEL=info
LOG_FORMAT=json
LOG_FILE_ENABLED=true
LOG_FILE_PATH=/var/log/cryptojackal/app.log

# --------------------------------------------------------------
# Monitoring Configuration
# --------------------------------------------------------------
METRICS_ENABLED=true
METRICS_PORT=9090
HEALTH_CHECK_ENABLED=true
HEALTH_CHECK_PORT=8081

# --------------------------------------------------------------
# Token Discovery Configuration
# --------------------------------------------------------------
DEXSCREENER_API_URL=https://api.dexscreener.com/latest/dex
COINGECKO_API_URL=https://api.coingecko.com/api/v3
DISCOVERY_SCAN_INTERVAL=30000
MAX_NEW_TOKENS_PER_SCAN=10
TOKEN_SECURITY_CHECK_ENABLED=true

# --------------------------------------------------------------
# API Keys (Optional)
# --------------------------------------------------------------
"#,
        config.environment,
        chrono::Utc::now().format("%Y-%m-%d %H:%M:%S UTC"),
        config.node_url,
        config.chain_id,
        config.scan_interval,
        config.gas_limit,
        config.slippage_tolerance,
        config.min_liquidity,
        config.trade_amount,
        config.paper_trading_mode,
        config.cors_origins,
        config.jwt_secret
    );

    // Add optional API keys
    if let Some(ref key) = config.coingecko_api_key {
        content.push_str(&format!("COINGECKO_API_KEY={}\n", key));
    }
    
    if let Some(ref key) = config.dexscreener_api_key {
        content.push_str(&format!("DEXSCREENER_API_KEY={}\n", key));
    }
    
    if let Some(ref token) = config.telegram_bot_token {
        content.push_str(&format!("TELEGRAM_BOT_TOKEN={}\n", token));
    }
    
    if let Some(ref url) = config.discord_webhook_url {
        content.push_str(&format!("DISCORD_WEBHOOK_URL={}\n", url));
    }

    // Add environment-specific settings
    content.push_str(&format!(
        r#"
# --------------------------------------------------------------
# {} Environment Settings
# --------------------------------------------------------------
ENVIRONMENT={}
DEBUG_MODE={}
HOT_RELOAD={}
ENABLE_PROFILING={}
"#,
        config.environment.to_uppercase(),
        config.environment,
        if config.environment == "production" { "false" } else { "true" },
        if config.environment == "production" { "false" } else { "true" },
        if config.environment == "production" { "true" } else { "false" }
    ));

    content
}

/// Deploy and start services
pub async fn deploy_setup(
    State(state): State<ApiState>,
    Json(config): Json<SetupConfig>,
) -> Result<Json<SetupResponse>, StatusCode> {
    info!("Starting deployment for environment: {}", config.environment);

    // First save configuration
    let save_result = save_config(State(state.clone()), Json(config.clone())).await?;
    if !save_result.success {
        return Ok(save_result);
    }

    // Start deployment based on environment
    let deployment_result = if config.environment == "test" {
        deploy_test_environment().await
    } else if config.environment == "production" {
        deploy_production_environment().await
    } else {
        deploy_development_environment().await
    };

    match deployment_result {
        Ok(_) => {
            info!("Deployment started successfully");
            Ok(Json(SetupResponse {
                success: true,
                message: "Deployment started successfully".to_string(),
                data: Some(serde_json::json!({
                    "environment": config.environment,
                    "status": "deploying",
                    "next_steps": [
                        "Wait for services to start",
                        "Access the web interface",
                        "Connect MetaMask wallet",
                        "Start with paper trading"
                    ]
                })),
            }))
        }
        Err(e) => {
            error!("Deployment failed: {}", e);
            Ok(Json(SetupResponse {
                success: false,
                message: format!("Deployment failed: {}", e),
                data: None,
            }))
        }
    }
}

/// Deploy test environment
async fn deploy_test_environment() -> Result<()> {
    info!("Deploying test environment");

    // Use docker-compose for test environment
    let output = Command::new("docker-compose")
        .args(&["-f", "docker-compose.dev.yml", "up", "-d"])
        .output()
        .await?;

    if !output.status.success() {
        let error = String::from_utf8_lossy(&output.stderr);
        return Err(anyhow::anyhow!("Docker compose failed: {}", error));
    }

    Ok(())
}

/// Deploy production environment
async fn deploy_production_environment() -> Result<()> {
    info!("Deploying production environment");

    // Use docker-compose for production
    let output = Command::new("docker-compose")
        .args(&["up", "-d"])
        .output()
        .await?;

    if !output.status.success() {
        let error = String::from_utf8_lossy(&output.stderr);
        return Err(anyhow::anyhow!("Docker compose failed: {}", error));
    }

    Ok(())
}

/// Deploy development environment
async fn deploy_development_environment() -> Result<()> {
    info!("Deploying development environment");

    // For development, we might want to start services individually
    // or provide instructions for manual startup
    
    Ok(())
}

/// Get deployment status
pub async fn deployment_status() -> Result<Json<DeploymentStatus>, StatusCode> {
    info!("Checking deployment status");

    let mut services = Vec::new();
    let mut overall_status = "unknown";
    let mut progress = 0;

    // Check if Docker services are running
    if let Ok(output) = Command::new("docker-compose")
        .args(&["ps", "--format", "json"])
        .output()
        .await
    {
        if output.status.success() {
            let output_str = String::from_utf8_lossy(&output.stdout);
            
            // Parse docker-compose output (simplified)
            if !output_str.trim().is_empty() {
                services.push(ServiceStatus {
                    name: "CryptoJackal Backend".to_string(),
                    status: "running".to_string(),
                    url: Some("http://localhost:8080".to_string()),
                });
                
                services.push(ServiceStatus {
                    name: "Web Frontend".to_string(),
                    status: "running".to_string(),
                    url: Some("http://localhost:3000".to_string()),
                });
                
                services.push(ServiceStatus {
                    name: "Database".to_string(),
                    status: "running".to_string(),
                    url: None,
                });
                
                progress = 100;
                overall_status = "running";
            }
        }
    }

    // If no services found, check if setup is in progress
    if services.is_empty() {
        services.push(ServiceStatus {
            name: "Setup".to_string(),
            status: "pending".to_string(),
            url: None,
        });
        progress = 0;
        overall_status = "pending";
    }

    Ok(Json(DeploymentStatus {
        status: overall_status.to_string(),
        message: format!("Deployment is {}", overall_status),
        progress,
        services,
    }))
}

/// Setup health check
pub async fn setup_health() -> Result<Json<SetupResponse>, StatusCode> {
    info!("Setup health check");

    let checks = vec![
        ("docker", check_docker().await),
        ("docker_compose", check_docker_compose().await),
        ("node", check_node().await),
        ("npm", check_npm().await),
    ];

    let mut failed_checks = Vec::new();

    for (name, passed) in checks {
        if !passed {
            failed_checks.push(name);
        }
    }

    if failed_checks.is_empty() {
        Ok(Json(SetupResponse {
            success: true,
            message: "All setup prerequisites satisfied".to_string(),
            data: Some(serde_json::json!({
                "status": "ready",
                "checks": checks
            })),
        }))
    } else {
        Ok(Json(SetupResponse {
            success: false,
            message: format!("Missing prerequisites: {}", failed_checks.join(", ")),
            data: Some(serde_json::json!({
                "status": "not_ready",
                "failed_checks": failed_checks
            })),
        }))
    }
}

/// Check if Docker is available
async fn check_docker() -> bool {
    Command::new("docker")
        .arg("--version")
        .output()
        .await
        .map(|output| output.status.success())
        .unwrap_or(false)
}

/// Check if Docker Compose is available
async fn check_docker_compose() -> bool {
    Command::new("docker-compose")
        .arg("--version")
        .output()
        .await
        .map(|output| output.status.success())
        .unwrap_or(false)
}

/// Check if Node.js is available
async fn check_node() -> bool {
    Command::new("node")
        .arg("--version")
        .output()
        .await
        .map(|output| output.status.success())
        .unwrap_or(false)
}

/// Check if npm is available
async fn check_npm() -> bool {
    Command::new("npm")
        .arg("--version")
        .output()
        .await
        .map(|output| output.status.success())
        .unwrap_or(false)
}
