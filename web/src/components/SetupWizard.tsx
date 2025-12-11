import React, { useState, useEffect } from 'react';
import { ChevronRight, ChevronLeft, Check, AlertTriangle, Settings, Database, Shield, Rocket, TestTube } from 'lucide-react';
import { toast } from 'react-hot-toast';

interface SetupConfig {
  environment: 'test' | 'production' | 'development';
  nodeUrl: string;
  chainId: string;
  tradeAmount: string;
  scanInterval: string;
  gasLimit: string;
  slippageTolerance: string;
  minLiquidity: string;
  paperTradingMode: boolean;
  coingeckoApiKey: string;
  dexscreenerApiKey: string;
  telegramBotToken: string;
  discordWebhookUrl: string;
  jwtSecret: string;
  corsOrigins: string;
}

const SetupWizard: React.FC = () => {
  const [currentStep, setCurrentStep] = useState(0);
  const [config, setConfig] = useState<SetupConfig>({
    environment: 'test',
    nodeUrl: '',
    chainId: '1',
    tradeAmount: '100000000000000000000',
    scanInterval: '1000',
    gasLimit: '300000',
    slippageTolerance: '0.005',
    minLiquidity: '10.0',
    paperTradingMode: true,
    coingeckoApiKey: '',
    dexscreenerApiKey: '',
    telegramBotToken: '',
    discordWebhookUrl: '',
    jwtSecret: '',
    corsOrigins: 'http://localhost:3000'
  });

  const [isSetupComplete, setIsSetupComplete] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  const steps = [
    { id: 'welcome', title: 'Welcome', icon: Rocket, description: 'Welcome to CryptoJackal Setup Wizard' },
    { id: 'environment', title: 'Environment', icon: TestTube, description: 'Choose your setup environment' },
    { id: 'node', title: 'Node Config', icon: Database, description: 'Configure Ethereum node connection' },
    { id: 'trading', title: 'Trading', icon: Settings, description: 'Set up trading parameters' },
    { id: 'api', title: 'API Keys', icon: Shield, description: 'Configure optional API keys' },
    { id: 'security', title: 'Security', icon: Shield, description: 'Security configuration' },
    { id: 'deploy', title: 'Deploy', icon: Rocket, description: 'Build and deploy your setup' }
  ];

  const validateStep = (stepId: string): boolean => {
    const errors: Record<string, string> = {};

    switch (stepId) {
      case 'environment':
        if (!config.environment) {
          errors.environment = 'Please select an environment';
        }
        break;

      case 'node':
        if (!config.nodeUrl) {
          errors.nodeUrl = 'Node URL is required';
        } else if (!config.nodeUrl.startsWith('http')) {
          errors.nodeUrl = 'Node URL must start with http:// or https://';
        }
        if (!config.chainId) {
          errors.chainId = 'Chain ID is required';
        }
        break;

      case 'trading':
        if (!config.tradeAmount || parseFloat(config.tradeAmount) <= 0) {
          errors.tradeAmount = 'Trade amount must be greater than 0';
        }
        if (!config.scanInterval || parseInt(config.scanInterval) <= 0) {
          errors.scanInterval = 'Scan interval must be greater than 0';
        }
        if (!config.gasLimit || parseInt(config.gasLimit) <= 0) {
          errors.gasLimit = 'Gas limit must be greater than 0';
        }
        break;

      case 'security':
        if (!config.jwtSecret) {
          errors.jwtSecret = 'JWT secret is required';
        } else if (config.jwtSecret.length < 32) {
          errors.jwtSecret = 'JWT secret must be at least 32 characters';
        }
        break;

      default:
        break;
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const nextStep = () => {
    const currentStepId = steps[currentStep].id;
    if (validateStep(currentStepId)) {
      if (currentStep < steps.length - 1) {
        setCurrentStep(currentStep + 1);
      }
    } else {
      toast.error('Please fix validation errors before proceeding');
    }
  };

  const prevStep = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1);
    }
  };

  const updateConfig = (key: keyof SetupConfig, value: any) => {
    setConfig(prev => ({ ...prev, [key]: value }));
    // Clear validation error for this field
    if (validationErrors[key]) {
      setValidationErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[key];
        return newErrors;
      });
    }
  };

  const generateJwtSecret = () => {
    const secret = Array.from(crypto.getRandomValues(new Uint8Array(32)))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');
    updateConfig('jwtSecret', secret);
    toast.success('JWT secret generated');
  };

  const saveConfiguration = async () => {
    setIsLoading(true);
    try {
      const response = await fetch('/api/setup/save-config', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(config),
      });

      if (response.ok) {
        toast.success('Configuration saved successfully');
        setIsSetupComplete(true);
      } else {
        throw new Error('Failed to save configuration');
      }
    } catch (error) {
      toast.error('Failed to save configuration');
      console.error('Setup error:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const deploySetup = async () => {
    setIsLoading(true);
    try {
      const response = await fetch('/api/setup/deploy', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(config),
      });

      if (response.ok) {
        toast.success('Deployment started successfully');
        setIsSetupComplete(true);
      } else {
        throw new Error('Failed to start deployment');
      }
    } catch (error) {
      toast.error('Failed to start deployment');
      console.error('Deployment error:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const renderStepContent = () => {
    const step = steps[currentStep];
    const Icon = step.icon;

    switch (step.id) {
      case 'welcome':
        return (
          <div className="text-center space-y-6">
            <div className="mx-auto w-20 h-20 bg-blue-100 rounded-full flex items-center justify-center">
              <Icon className="w-10 h-10 text-blue-600" />
            </div>
            <h2 className="text-2xl font-bold text-gray-900">Welcome to CryptoJackal</h2>
            <p className="text-gray-600 max-w-md mx-auto">
              This wizard will guide you through setting up CryptoJackal from test environment to production deployment.
            </p>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 max-w-md mx-auto">
              <h3 className="font-semibold text-blue-900 mb-2">What you'll need:</h3>
              <ul className="text-sm text-blue-800 space-y-1 text-left">
                <li>• Ethereum node access (Infura, Alchemy, or custom)</li>
                <li>• MetaMask wallet for secure trading</li>
                <li>• Optional: API keys for enhanced features</li>
              </ul>
            </div>
          </div>
        );

      case 'environment':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <Icon className="w-12 h-12 text-blue-600 mx-auto mb-4" />
              <h2 className="text-xl font-bold text-gray-900">Choose Environment</h2>
              <p className="text-gray-600">Select your setup environment</p>
            </div>

            <div className="space-y-3">
              {[
                { value: 'test', label: 'Test Environment', desc: 'Safe testing with paper trading', color: 'green' },
                { value: 'development', label: 'Development', desc: 'For developers and testing', color: 'yellow' },
                { value: 'production', label: 'Production', desc: 'Live trading with real funds', color: 'red' }
              ].map((env) => (
                <button
                  key={env.value}
                  onClick={() => updateConfig('environment', env.value as any)}
                  className={`w-full p-4 rounded-lg border-2 text-left transition-all ${
                    config.environment === env.value
                      ? `border-${env.color}-500 bg-${env.color}-50`
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  <div className="font-semibold text-gray-900">{env.label}</div>
                  <div className="text-sm text-gray-600">{env.desc}</div>
                </button>
              ))}
            </div>

            {validationErrors.environment && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-3">
                <div className="flex items-center text-red-800">
                  <AlertTriangle className="w-4 h-4 mr-2" />
                  {validationErrors.environment}
                </div>
              </div>
            )}
          </div>
        );

      case 'node':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <Icon className="w-12 h-12 text-blue-600 mx-auto mb-4" />
              <h2 className="text-xl font-bold text-gray-900">Node Configuration</h2>
              <p className="text-gray-600">Configure your Ethereum node connection</p>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Node Provider
                </label>
                <select
                  value={config.nodeUrl.includes('infura') ? 'infura' : 
                         config.nodeUrl.includes('alchemy') ? 'alchemy' : 
                         config.nodeUrl.includes('quiknode') ? 'quiknode' : 'custom'}
                  onChange={(e) => {
                    const provider = e.target.value;
                    const urls = {
                      infura: 'https://mainnet.infura.io/v3/YOUR_PROJECT_ID',
                      alchemy: 'https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY',
                      quiknode: 'https://YOUR_ENDPOINT.quiknode.pro/YOUR_KEY',
                      custom: ''
                    };
                    updateConfig('nodeUrl', urls[provider as keyof typeof urls]);
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="infura">Infura</option>
                  <option value="alchemy">Alchemy</option>
                  <option value="quiknode">QuickNode</option>
                  <option value="custom">Custom</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Node URL
                </label>
                <input
                  type="url"
                  value={config.nodeUrl}
                  onChange={(e) => updateConfig('nodeUrl', e.target.value)}
                  placeholder="https://mainnet.infura.io/v3/YOUR_PROJECT_ID"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                {validationErrors.nodeUrl && (
                  <p className="text-red-600 text-sm mt-1">{validationErrors.nodeUrl}</p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Chain ID
                </label>
                <input
                  type="text"
                  value={config.chainId}
                  onChange={(e) => updateConfig('chainId', e.target.value)}
                  placeholder="1"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                {validationErrors.chainId && (
                  <p className="text-red-600 text-sm mt-1">{validationErrors.chainId}</p>
                )}
              </div>
            </div>
          </div>
        );

      case 'trading':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <Icon className="w-12 h-12 text-blue-600 mx-auto mb-4" />
              <h2 className="text-xl font-bold text-gray-900">Trading Configuration</h2>
              <p className="text-gray-600">Set up your trading parameters</p>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Trade Amount (wei)
                </label>
                <input
                  type="text"
                  value={config.tradeAmount}
                  onChange={(e) => updateConfig('tradeAmount', e.target.value)}
                  placeholder="100000000000000000000"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Common amounts: 0.1 ETH = 100000000000000000000 wei
                </p>
                {validationErrors.tradeAmount && (
                  <p className="text-red-600 text-sm mt-1">{validationErrors.tradeAmount}</p>
                )}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Scan Interval (ms)
                  </label>
                  <input
                    type="number"
                    value={config.scanInterval}
                    onChange={(e) => updateConfig('scanInterval', e.target.value)}
                    placeholder="1000"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                  {validationErrors.scanInterval && (
                    <p className="text-red-600 text-sm mt-1">{validationErrors.scanInterval}</p>
                  )}
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Gas Limit
                  </label>
                  <input
                    type="number"
                    value={config.gasLimit}
                    onChange={(e) => updateConfig('gasLimit', e.target.value)}
                    placeholder="300000"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                  {validationErrors.gasLimit && (
                    <p className="text-red-600 text-sm mt-1">{validationErrors.gasLimit}</p>
                  )}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Slippage Tolerance
                  </label>
                  <input
                    type="text"
                    value={config.slippageTolerance}
                    onChange={(e) => updateConfig('slippageTolerance', e.target.value)}
                    placeholder="0.005"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                  <p className="text-xs text-gray-500 mt-1">0.005 = 0.5%</p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Min Liquidity (ETH)
                  </label>
                  <input
                    type="text"
                    value={config.minLiquidity}
                    onChange={(e) => updateConfig('minLiquidity', e.target.value)}
                    placeholder="10.0"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
              </div>
            </div>
          </div>
        );

      case 'api':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <Icon className="w-12 h-12 text-blue-600 mx-auto mb-4" />
              <h2 className="text-xl font-bold text-gray-900">API Keys (Optional)</h2>
              <p className="text-gray-600">Enhance features with API keys</p>
            </div>

            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <p className="text-sm text-blue-800">
                API keys are optional but provide enhanced features like better token discovery and notifications.
              </p>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  CoinGecko API Key
                </label>
                <input
                  type="password"
                  value={config.coingeckoApiKey}
                  onChange={(e) => updateConfig('coingeckoApiKey', e.target.value)}
                  placeholder="Your CoinGecko API key"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <p className="text-xs text-gray-500 mt-1">Free tier available at coingecko.com</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  DexScreener API Key
                </label>
                <input
                  type="password"
                  value={config.dexscreenerApiKey}
                  onChange={(e) => updateConfig('dexscreenerApiKey', e.target.value)}
                  placeholder="Your DexScreener API key"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Telegram Bot Token
                </label>
                <input
                  type="password"
                  value={config.telegramBotToken}
                  onChange={(e) => updateConfig('telegramBotToken', e.target.value)}
                  placeholder="Your Telegram bot token"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Discord Webhook URL
                </label>
                <input
                  type="url"
                  value={config.discordWebhookUrl}
                  onChange={(e) => updateConfig('discordWebhookUrl', e.target.value)}
                  placeholder="Your Discord webhook URL"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
          </div>
        );

      case 'security':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <Icon className="w-12 h-12 text-blue-600 mx-auto mb-4" />
              <h2 className="text-xl font-bold text-gray-900">Security Configuration</h2>
              <p className="text-gray-600">Configure security settings</p>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  JWT Secret
                </label>
                <div className="flex gap-2">
                  <input
                    type="password"
                    value={config.jwtSecret}
                    onChange={(e) => updateConfig('jwtSecret', e.target.value)}
                    placeholder="Secure JWT secret"
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                  <button
                    onClick={generateJwtSecret}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                  >
                    Generate
                  </button>
                </div>
                {validationErrors.jwtSecret && (
                  <p className="text-red-600 text-sm mt-1">{validationErrors.jwtSecret}</p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  CORS Origins
                </label>
                <input
                  type="text"
                  value={config.corsOrigins}
                  onChange={(e) => updateConfig('corsOrigins', e.target.value)}
                  placeholder="http://localhost:3000"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <p className="text-xs text-gray-500 mt-1">Comma-separated origins</p>
              </div>

              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                <div className="flex items-center text-yellow-800">
                  <AlertTriangle className="w-4 h-4 mr-2" />
                  <span className="font-semibold">Security Note</span>
                </div>
                <p className="text-sm text-yellow-700 mt-1">
                  Use a strong, unique JWT secret. Never share it or commit it to version control.
                </p>
              </div>
            </div>
          </div>
        );

      case 'deploy':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <Icon className="w-12 h-12 text-blue-600 mx-auto mb-4" />
              <h2 className="text-xl font-bold text-gray-900">Deploy Setup</h2>
              <p className="text-gray-600">Review and deploy your configuration</p>
            </div>

            <div className="bg-gray-50 rounded-lg p-4">
              <h3 className="font-semibold text-gray-900 mb-3">Configuration Summary</h3>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-gray-600">Environment:</span>
                  <span className="font-medium capitalize">{config.environment}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Chain ID:</span>
                  <span className="font-medium">{config.chainId}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Trade Amount:</span>
                  <span className="font-medium">{config.tradeAmount} wei</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Paper Trading:</span>
                  <span className="font-medium">{config.paperTradingMode ? 'Enabled' : 'Disabled'}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">API Keys Configured:</span>
                  <span className="font-medium">
                    {[config.coingeckoApiKey, config.dexscreenerApiKey, config.telegramBotToken, config.discordWebhookUrl]
                      .filter(Boolean).length > 0 ? 'Yes' : 'No'}
                  </span>
                </div>
              </div>
            </div>

            {config.environment === 'production' && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                <div className="flex items-center text-red-800">
                  <AlertTriangle className="w-4 h-4 mr-2" />
                  <span className="font-semibold">Production Warning</span>
                </div>
                <p className="text-sm text-red-700 mt-1">
                  You are deploying to production. Ensure all configurations are correct and you have proper security measures in place.
                </p>
              </div>
            )}

            <div className="flex flex-col gap-3">
              <button
                onClick={saveConfiguration}
                disabled={isLoading}
                className="w-full px-4 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
              >
                {isLoading ? 'Saving...' : 'Save Configuration'}
              </button>

              <button
                onClick={deploySetup}
                disabled={isLoading}
                className="w-full px-4 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors disabled:opacity-50"
              >
                {isLoading ? 'Deploying...' : 'Deploy & Start'}
              </button>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  if (isSetupComplete) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8 text-center">
          <div className="mx-auto w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mb-4">
            <Check className="w-8 h-8 text-green-600" />
          </div>
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Setup Complete!</h2>
          <p className="text-gray-600 mb-6">
            CryptoJackal has been configured and deployed successfully.
          </p>
          <div className="space-y-3">
            <a
              href="/"
              className="block w-full px-4 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-center"
            >
              Go to Dashboard
            </a>
            <a
              href="/settings"
              className="block w-full px-4 py-3 bg-gray-200 text-gray-800 rounded-lg hover:bg-gray-300 transition-colors text-center"
            >
              Manage Settings
            </a>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-4xl mx-auto p-8">
        {/* Progress Bar */}
        <div className="mb-8">
          <div className="flex items-center justify-between mb-4">
            {steps.map((step, index) => {
              const StepIcon = step.icon;
              const isActive = index === currentStep;
              const isCompleted = index < currentStep;
              
              return (
                <div key={step.id} className="flex items-center">
                  <div
                    className={`w-10 h-10 rounded-full flex items-center justify-center transition-colors ${
                      isActive
                        ? 'bg-blue-600 text-white'
                        : isCompleted
                        ? 'bg-green-600 text-white'
                        : 'bg-gray-200 text-gray-600'
                    }`}
                  >
                    {isCompleted ? (
                      <Check className="w-5 h-5" />
                    ) : (
                      <StepIcon className="w-5 h-5" />
                    )}
                  </div>
                  <div className="ml-3">
                    <div className={`text-sm font-medium ${
                      isActive ? 'text-blue-600' : isCompleted ? 'text-green-600' : 'text-gray-600'
                    }`}>
                      {step.title}
                    </div>
                    <div className="text-xs text-gray-500">{step.description}</div>
                  </div>
                  {index < steps.length - 1 && (
                    <div className={`w-full h-0.5 mx-4 ${
                      index < currentStep ? 'bg-green-600' : 'bg-gray-200'
                    }`} />
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* Step Content */}
        <div className="bg-white rounded-lg shadow-lg p-8">
          {renderStepContent()}

          {/* Navigation */}
          <div className="flex justify-between mt-8">
            <button
              onClick={prevStep}
              disabled={currentStep === 0}
              className="flex items-center px-4 py-2 text-gray-600 hover:text-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="w-4 h-4 mr-2" />
              Previous
            </button>

            <button
              onClick={nextStep}
              disabled={currentStep === steps.length - 1}
              className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next
              <ChevronRight className="w-4 h-4 ml-2" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SetupWizard;
