# Junior Developer Security & Coordination Guidelines

## 🚨 **CRITICAL SECURITY REQUIREMENTS**

### **ABSOLUTE PROHIBITIONS**
- ❌ **NEVER** add `private_key`, `secret_key`, or any private key fields to ANY struct
- ❌ **NEVER** store private keys in configuration, environment variables, or code
- ❌ **NEVER** create wallet functionality that handles private keys directly
- ❌ **NEVER** implement signing logic that bypasses MetaMask

### **MANDATORY SECURITY PRINCIPLES**
- ✅ **MetaMask-ONLY** wallet integration - all signing delegated to MetaMask
- ✅ **Zero private key storage** anywhere in the codebase
- ✅ **Event-driven architecture** for wallet interactions
- ✅ **Async-first design** for all modules

## 📋 **PRE-IMPLEMENTATION CHECKLIST**

### **Before Writing ANY Code:**
1. **Review Senior's Latest Work**
   - Check `src/core/mod.rs` for Bot struct changes
   - Review `src/wallet/mod.rs` for MetaMask integration patterns
   - Check `src/trading/mod.rs` for trading logic integration
   - Review `Cargo.toml` for latest dependencies

2. **Check Existing Implementations**
   - Search for similar functionality already implemented
   - Review integration patterns with order queue system
   - Check error handling patterns in existing modules

3. **Validate Security Compliance**
   - Ensure no private key storage anywhere
   - Confirm MetaMask-only wallet interactions
   - Verify async patterns match existing code

4. **Coordinate with Senior**
   - Check if module affects shared components (Config, Bot, etc.)
   - Verify integration points with order queue system
   - Confirm task assignment is still current

## 🔧 **IMPLEMENTATION STANDARDS**

### **Configuration Module Standards**
```rust
// ✅ CORRECT - No private keys
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Config {
    pub node_url: String,
    pub websocket_url: Option<String>,
    // ... other config fields
    // NO PRIVATE KEY FIELDS!
}

// ❌ WRONG - Private key storage
pub struct Config {
    pub private_key: String,  // SECURITY VIOLATION!
}
```

### **Wallet Integration Standards**
```rust
// ✅ CORRECT - MetaMask delegation
pub async fn sign_transaction(&self, tx: TransactionRequest) -> Result<String> {
    // Delegate to MetaMask
    self.request_metamask_signature(tx).await
}

// ❌ WRONG - Direct private key usage
pub async fn sign_transaction(&self, tx: TransactionRequest, private_key: &str) -> Result<String> {
    // SECURITY VIOLATION!
}
```

## 🔄 **COORDINATION WORKFLOW**

### **Task Startup Protocol**
1. **Sync with latest code** from senior's implementations
2. **Review task assignment** in `templates/current_tasks.md`
3. **Check integration requirements** with existing modules
4. **Validate security compliance** before starting

### **During Implementation**
1. **Follow existing patterns** from senior's code
2. **Maintain async-first design** throughout
3. **Use established error handling** patterns
4. **Integrate with order queue system** where applicable

### **Before Completion**
1. **Security review** - no private key violations
2. **Integration test** with existing modules
3. **Code review** against senior's patterns
4. **Update task status** in templates

## 📚 **LEARNING FROM MISTAKES**

### **Recent Security Violation Analysis**
**Issue**: Config struct included `private_key` field
**Root Cause**: Didn't review existing MetaMask-only requirements
**Prevention**: Always check security requirements before implementation
**Learning**: Security is non-negotiable - when in doubt, ask senior

### **Integration Coordination Lessons**
**Issue**: Missing order queue configuration integration
**Root Cause**: Didn't review senior's latest order queue implementation
**Prevention**: Always sync with senior's latest work before shared modules
**Learning**: Shared modules require coordination - check existing implementations

## 🎯 **SUCCESS METRICS**

### **Security Compliance**
- [ ] Zero private key references in code
- [ ] MetaMask-only wallet integration
- [ ] No security violations requiring senior fixes

### **Coordination Excellence**
- [ ] Code integrates seamlessly with senior's work
- [ ] No conflicts with existing implementations
- [ ] Follows established patterns and conventions

### **Quality Standards**
- [ ] Async-first design maintained
- [ ] Error handling follows project patterns
- [ ] Documentation and tests included

## 🚀 **NEXT STEPS FOR JUNIOR**

1. **Review this document** before every task
2. **Study senior's implementations** as reference patterns
3. **Ask questions** when security requirements are unclear
4. **Coordinate proactively** on shared modules
5. **Learn from mistakes** to prevent repetition

---

**Remember: Security violations are learning opportunities, but they should not repeat. The goal is proactive compliance, not reactive fixes.**
