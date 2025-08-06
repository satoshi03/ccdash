#!/usr/bin/env node

const { spawn, exec } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

// ASCII Art Logo
const LOGO = `
 ██████╗ ██████╗██████╗  █████╗ ███████╗██╗  ██╗
██╔════╝██╔════╝██╔══██╗██╔══██╗██╔════╝██║  ██║
██║     ██║     ██║  ██║███████║███████╗███████║
██║     ██║     ██║  ██║██╔══██║╚════██║██╔══██║
╚██████╗╚██████╗██████╔╝██║  ██║███████║██║  ██║
 ╚═════╝ ╚═════╝╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝

            Claude Code Dashboard
`;

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m',
};

const log = {
  info: (msg) => console.log(`${colors.blue}ℹ${colors.reset} ${msg}`),
  success: (msg) => console.log(`${colors.green}✓${colors.reset} ${msg}`),
  warning: (msg) => console.log(`${colors.yellow}⚠${colors.reset} ${msg}`),
  error: (msg) => console.log(`${colors.red}✗${colors.reset} ${msg}`),
  logo: () => console.log(`${colors.cyan}${LOGO}${colors.reset}`),
};

// Get the package root directory
const packageRoot = path.dirname(__dirname);
const backendPath = path.join(packageRoot, 'backend');

// Set frontend path based on environment
let frontendPath;
if (isNpmPackage()) {
  // For npm packages, frontend is in bin/frontend-dist
  frontendPath = path.join(packageRoot, 'bin', 'frontend-dist');
} else {
  // For development, frontend is in frontend/
  frontendPath = path.join(packageRoot, 'frontend');
}

// Browser launch helper function
function openBrowser(url) {
  let command;
  
  switch (process.platform) {
    case 'darwin': // macOS
      command = `open "${url}"`;
      break;
    case 'win32': // Windows
      command = `start "" "${url}"`;
      break;
    default: // Linux and others
      command = `xdg-open "${url}"`;
      break;
  }
  
  exec(command, (error) => {
    if (error) {
      log.warning(`Could not open browser automatically: ${error.message}`);
      log.info(`Please open your browser and navigate to: ${url}`);
    } else {
      log.success(`Opened browser at: ${url}`);
    }
  });
}

// Wait for service to be ready before opening browser
function waitForService(url, maxAttempts = 30, interval = 1000) {
  return new Promise((resolve, reject) => {
    let attempts = 0;
    
    const checkService = () => {
      attempts++;
      
      // Simple HTTP check using curl or similar
      const testCommand = process.platform === 'win32' 
        ? `powershell -Command "try { Invoke-WebRequest -Uri '${url}' -TimeoutSec 1 -UseBasicParsing | Out-Null; exit 0 } catch { exit 1 }"`
        : `curl -s --connect-timeout 1 --max-time 1 "${url}" > /dev/null 2>&1`;
      
      exec(testCommand, (error) => {
        if (!error) {
          resolve(true);
        } else if (attempts >= maxAttempts) {
          reject(new Error(`Service not ready after ${maxAttempts} attempts`));
        } else {
          setTimeout(checkService, interval);
        }
      });
    };
    
    checkService();
  });
}

// Early function declaration needed for path setup
function isNpmPackage() {
  // Check for npm environment variables
  if (process.env.npm_package_name === 'ccdash' ||
      process.env.npm_config_global === 'true') {
    return true;
  }

  // Check if we're in a node_modules directory
  if (packageRoot.includes('node_modules')) {
    return true;
  }

  // Check if we don't have development files (indicating we're a packaged install)
  return !fs.existsSync(path.join(packageRoot, '.git')) &&
         !fs.existsSync(path.join(packageRoot, 'go.mod')) &&
         !fs.existsSync(path.join(packageRoot, 'backend'));
}


// Check if Go is installed
function checkGoInstallation() {
  return new Promise((resolve) => {
    const goCheck = spawn('go', ['version'], { stdio: 'ignore' });
    goCheck.on('close', (code) => {
      resolve(code === 0);
    });
    goCheck.on('error', () => {
      resolve(false);
    });
  });
}

// Check if Node.js dependencies are installed
function checkNodeDependencies() {
  const nodeModulesPath = path.join(frontendPath, 'node_modules');
  return fs.existsSync(nodeModulesPath);
}

// Build backend if needed
async function buildBackend() {
  return new Promise((resolve, reject) => {
    log.info('Building backend server...');

    const buildProcess = spawn('go', ['build', '-o', path.join(packageRoot, 'bin', 'ccdash-server'), 'cmd/server/main.go'], {
      cwd: backendPath,
      stdio: 'inherit'
    });

    buildProcess.on('close', (code) => {
      if (code === 0) {
        log.success('Backend built successfully');
        resolve();
      } else {
        reject(new Error(`Backend build failed with code ${code}`));
      }
    });

    buildProcess.on('error', (err) => {
      reject(new Error(`Failed to build backend: ${err.message}`));
    });
  });
}

// Install frontend dependencies if needed
async function installFrontendDependencies() {
  return new Promise((resolve, reject) => {
    log.info('Installing frontend dependencies...');

    const installProcess = spawn('npm', ['install', '--legacy-peer-deps'], {
      cwd: frontendPath,
      stdio: 'inherit'
    });

    installProcess.on('close', (code) => {
      if (code === 0) {
        log.success('Frontend dependencies installed');
        resolve();
      } else {
        reject(new Error(`Frontend dependency installation failed with code ${code}`));
      }
    });

    installProcess.on('error', (err) => {
      reject(new Error(`Failed to install frontend dependencies: ${err.message}`));
    });
  });
}

// Build frontend
async function buildFrontend() {
  return buildFrontendWithApiUrl(null);
}

// Build frontend with custom API URL
async function buildFrontendWithApiUrl(apiUrl) {
  return new Promise((resolve, reject) => {
    log.info('Building frontend...');

    const env = {
      ...process.env
    };
    
    if (apiUrl) {
      env.NEXT_PUBLIC_API_URL = apiUrl;
      log.info(`Setting NEXT_PUBLIC_API_URL to: ${apiUrl}`);
      
      // Clean the build directory to ensure fresh build with new API URL
      const nextDir = path.join(frontendPath, '.next');
      if (fs.existsSync(nextDir)) {
        log.info('Cleaning previous build to ensure fresh build with new API URL...');
        fs.rmSync(nextDir, { recursive: true, force: true });
      }
    }

    const buildProcess = spawn('npm', ['run', 'build'], {
      cwd: frontendPath,
      stdio: 'inherit',
      env: env
    });

    buildProcess.on('close', (code) => {
      if (code === 0) {
        log.success('Frontend built successfully');
        resolve();
      } else {
        log.warning('Frontend build completed with warnings');
        resolve(); // Continue even if there are warnings
      }
    });

    buildProcess.on('error', (err) => {
      reject(new Error(`Failed to build frontend: ${err.message}`));
    });
  });
}


// Parse command line arguments
function parseArgs() {
  const args = process.argv.slice(2);
  let command = 'start';
  const backendPort = 6060; // Fixed backend port
  let frontendPort = 3000;
  const backendUrl = null; // Backend URL is not configurable
  let frontendUrl = null;
  let openBrowserFlag = true; // Default: open browser automatically
  let disableSafetyCheck = false;
  let disableAuth = false;
  let apiKey = null;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (arg === '--backend-port' || arg === '-bp') {
      log.warning('Backend port is fixed at 6060 and cannot be changed');
      i++; // Skip the next argument (port value)
    } else if (arg === '--frontend-port' || arg === '-fp') {
      frontendPort = parseInt(args[i + 1]) || 3000;
      i++;
    } else if (arg === '--backend-url' || arg === '-bu') {
      log.warning('Backend URL is not configurable in npm package mode');
      i++; // Skip the next argument (URL value)
    } else if (arg === '--frontend-url' || arg === '-fu') {
      frontendUrl = args[i + 1];
      i++;
    } else if (arg === '--no-open' || arg === '--no-browser') {
      openBrowserFlag = false;
    } else if (arg === '--open' || arg === '--browser') {
      openBrowserFlag = true;
    } else if (arg === '--no-safety' || arg === '--disable-safety-check') {
      disableSafetyCheck = true;
    } else if (arg === '--no-auth' || arg === '--disable-auth') {
      disableAuth = true;
    } else if (arg === '--api-key' || arg === '-k') {
      apiKey = args[i + 1];
      if (!apiKey || apiKey.startsWith('-')) {
        log.error('API key value is required after --api-key');
        process.exit(1);
      }
      i++;
    } else if (arg.startsWith('--api-key=')) {
      apiKey = arg.split('=', 2)[1];
      if (!apiKey) {
        log.error('API key value is required after --api-key=');
        process.exit(1);
      }
    } else if (!arg.startsWith('-')) {
      command = arg;
    }
  }

  // Extract port from URL if provided
  if (frontendUrl) {
    try {
      const url = new URL(frontendUrl);
      frontendPort = parseInt(url.port) || (url.protocol === 'https:' ? 443 : 80);
    } catch (error) {
      log.warning(`Invalid frontend URL: ${frontendUrl}. Using default port ${frontendPort}`);
    }
  }

  return { command, backendPort, frontendPort, backendUrl, frontendUrl, openBrowser: openBrowserFlag, disableSafetyCheck, disableAuth, apiKey };
}

// Start backend server with fixed port 6060
function startBackend(port = 6060, frontendPort = 3000, frontendUrl = null, backendUrl = null, disableSafetyCheck = false, disableAuth = false, apiKey = null) {
  const serverPath = path.join(packageRoot, 'bin', 'ccdash-server');

  if (!fs.existsSync(serverPath)) {
    log.error('Backend server not found. Please run build first.');
    process.exit(1);
  }

  const frontendTargetUrl = frontendUrl || `http://localhost:${frontendPort}`;
  
  // Extract host from backend URL if provided
  let backendHost = 'localhost';
  if (backendUrl) {
    try {
      const url = new URL(backendUrl);
      backendHost = url.hostname;
    } catch (error) {
      log.warning(`Invalid backend URL: ${backendUrl}. Using localhost`);
    }
  }
  
  const backendDisplayUrl = backendUrl || `http://${backendHost}:${port}`;
  
  // Show warnings for disabled security features
  if (disableSafetyCheck) {
    log.warning('⚠️  Command safety checks are DISABLED - all commands will execute without validation!');
  }
  if (disableAuth) {
    log.warning('⚠️  API authentication is DISABLED - no API key required!');
  }
  
  log.info(`Starting backend server on ${backendDisplayUrl}`);
  
  const backendEnv = {
    ...process.env,
    PORT: port.toString(),
    HOST: backendHost,
    FRONTEND_URL: frontendTargetUrl
  };
  
  // Apply CLI overrides
  if (disableSafetyCheck) {
    backendEnv.COMMAND_WHITELIST_ENABLED = 'false';
  }
  if (disableAuth) {
    backendEnv.CCDASH_API_KEY = ''; // Clear API key to disable auth
    backendEnv.GIN_MODE = 'debug'; // Force debug mode to disable auth
  } else if (apiKey) {
    // Set API key from command line
    backendEnv.CCDASH_API_KEY = apiKey;
    log.info(`🔑 Using API key from command line`);
  }
  
  const backendProcess = spawn(serverPath, [], {
    stdio: 'inherit',
    detached: false,
    env: backendEnv
  });

  return backendProcess;
}

// Start frontend server with custom port or URL
function startFrontend(port = 3000, backendPort = 6060, backendUrl = null, frontendUrl = null) {
  // Extract hostname from frontend URL for binding
  let hostname = 'localhost';
  if (frontendUrl) {
    try {
      const url = new URL(frontendUrl);
      hostname = url.hostname;
    } catch (error) {
      log.warning(`Invalid frontend URL: ${frontendUrl}. Using localhost`);
    }
  }
  
  const displayUrl = frontendUrl || `http://${hostname}:${port}`;
  log.info(`Starting frontend server on ${displayUrl}`);

  // Check if frontend directory exists
  if (!fs.existsSync(frontendPath)) {
    log.warning('Frontend not available in this installation');
    const backendTargetUrl = `http://localhost:6060`;
    log.info(`API server is running on ${backendTargetUrl}`);
    return null;
  }

  const nextDir = path.join(frontendPath, '.next');
  const standalonePath = path.join(frontendPath, '.next', 'standalone', 'server.js');
  const apiUrl = `http://localhost:6060`; // Fixed backend URL

  // For npm packages (bin/frontend-dist structure)
  if (isNpmPackage()) {
    const serverJsPath = path.join(frontendPath, 'server.js');

    // Check if this is a standalone build (has server.js in root)
    if (fs.existsSync(serverJsPath)) {
      log.info('Using pre-built standalone Next.js frontend from npm package');
      const frontendProcess = spawn('node', ['server.js'], {
        cwd: frontendPath,
        stdio: 'inherit',
        detached: false,
        env: {
          ...process.env,
          PORT: port.toString(),
          NEXT_PUBLIC_API_URL: `http://localhost:6060/api`,
          HOSTNAME: hostname === 'localhost' ? '0.0.0.0' : hostname
        }
      });
      return frontendProcess;
    }
    // Try nested standalone build
    else if (fs.existsSync(nextDir) && fs.existsSync(standalonePath)) {
      log.info('Using nested standalone build');
      const standaloneDir = path.dirname(standalonePath);
      const frontendProcess = spawn('node', ['server.js'], {
        cwd: standaloneDir,
        stdio: 'inherit',
        detached: false,
        env: {
          ...process.env,
          PORT: port.toString(),
          NEXT_PUBLIC_API_URL: `http://localhost:6060/api`,
          HOSTNAME: hostname === 'localhost' ? '0.0.0.0' : hostname
        }
      });
      return frontendProcess;
    }
    // Fallback to npm start
    else if (fs.existsSync(nextDir)) {
      log.info('Using npm start for pre-built frontend');
      const frontendProcess = spawn('npm', ['start'], {
        cwd: frontendPath,
        stdio: 'inherit',
        detached: false,
        env: {
          ...process.env,
          PORT: port.toString(),
          NEXT_PUBLIC_API_URL: `${apiUrl}/api`
        }
      });
      return frontendProcess;
    } else {
      log.warning('Frontend build not found in npm package');
      log.info(`API server is running on ${apiUrl}`);
      return null;
    }
  }

  // For development environments (original frontend/ structure)
  else {
    if (fs.existsSync(standalonePath)) {
      // Use standalone build
      log.info('Using standalone Next.js build');
      const standaloneDir = path.dirname(standalonePath);
      const frontendProcess = spawn('node', ['server.js'], {
        cwd: standaloneDir,
        stdio: 'inherit',
        detached: false,
        env: {
          ...process.env,
          PORT: port.toString(),
          NEXT_PUBLIC_API_URL: `http://localhost:6060/api`,
          HOSTNAME: hostname === 'localhost' ? '0.0.0.0' : hostname
        }
      });
      return frontendProcess;
    } else {
      // Fallback to npm start for development
      log.info('Using npm start (development mode)');
      const frontendProcess = spawn('npm', ['run', 'start'], {
        cwd: frontendPath,
        stdio: 'inherit',
        detached: false,
        env: {
          ...process.env,
          PORT: port.toString(),
          HOSTNAME: hostname === 'localhost' ? '0.0.0.0' : hostname,
          NEXT_PUBLIC_API_URL: `${apiUrl}/api`
        }
      });
      return frontendProcess;
    }
  }
}

// Main CLI function
async function main() {
  const { command, backendPort, frontendPort, backendUrl, frontendUrl, openBrowser: shouldOpenBrowser, disableSafetyCheck, disableAuth, apiKey } = parseArgs();

  log.logo();

  try {
    switch (command) {
      case 'start':
      case 'run':
        log.info('Starting ccdash...');

        // Check prerequisites
        const hasGo = await checkGoInstallation();
        if (!hasGo) {
          log.error('Go is not installed. Please install Go 1.21 or later.');
          log.info('Visit https://golang.org/dl/ to download Go');
          process.exit(1);
        }

        // Build backend if needed
        const serverPath = path.join(packageRoot, 'bin', 'ccdash-server');
        if (!fs.existsSync(serverPath)) {
          await buildBackend();
        }

        // Skip frontend setup for npm packages
        if (!isNpmPackage()) {
          // Install frontend dependencies if needed
          if (!checkNodeDependencies()) {
            await installFrontendDependencies();
          }

          // Build frontend if standalone build doesn't exist
          const standalonePath = path.join(frontendPath, '.next', 'standalone', 'server.js');
          const needsRebuild = !fs.existsSync(standalonePath);
          
          if (needsRebuild) {
            await buildFrontendWithApiUrl('http://localhost:6060/api');
          }
        } else {
          log.info('Running from npm package, skipping frontend build');
        }

        // Start both services
        const backendProcess = startBackend(backendPort, frontendPort, frontendUrl, backendUrl, disableSafetyCheck, disableAuth, apiKey);
        const frontendProcess = startFrontend(frontendPort, backendPort, backendUrl, frontendUrl);

        // Handle process cleanup
        const cleanup = () => {
          log.info('Shutting down ccdash...');
          if (backendProcess && !backendProcess.killed) {
            backendProcess.kill('SIGTERM');
          }
          if (frontendProcess && !frontendProcess.killed) {
            frontendProcess.kill('SIGTERM');
          }
          process.exit(0);
        };

        process.on('SIGINT', cleanup);
        process.on('SIGTERM', cleanup);

        log.success('ccdash is running!');
        const backendDisplayUrl = `http://localhost:6060`;
        const frontendDisplayUrl = frontendUrl || `http://localhost:${frontendPort}`;
        log.info(`Backend:  ${backendDisplayUrl}`);
        if (frontendProcess) {
          log.info(`Frontend: ${frontendDisplayUrl}`);
        }
        log.info('Press Ctrl+C to stop');

        // Open browser automatically if frontend is available and flag is set
        if (frontendProcess && shouldOpenBrowser) {
          log.info('Waiting for services to be ready...');
          
          // Wait for services in background to avoid blocking the main process
          (async () => {
            try {
              // Wait for both backend and frontend to be ready
              await Promise.all([
                waitForService(backendDisplayUrl + '/api/health'),
                waitForService(frontendDisplayUrl)
              ]);
              
              // Open browser after services are ready
              setTimeout(() => {
                openBrowser(frontendDisplayUrl);
              }, 1000); // Small delay to ensure services are fully ready
              
            } catch (error) {
              log.warning('Services took too long to start, please open browser manually');
              log.info(`Frontend: ${frontendDisplayUrl}`);
            }
          })();
        } else if (!frontendProcess && shouldOpenBrowser) {
          // If no frontend but browser should open, suggest API endpoint
          log.info('Frontend not available, but you can access the API directly');
          log.info(`API endpoint: ${backendDisplayUrl}/api/health`);
        }

        break;

      case 'build':
        log.info('Building ccdash...');
        await buildBackend();

        if (!isNpmPackage()) {
          if (!checkNodeDependencies()) {
            await installFrontendDependencies();
          }
          await buildFrontendWithApiUrl('http://localhost:6060/api');
        } else {
          log.info('Running from npm package, skipping frontend build');
        }

        log.success('Build completed successfully!');
        break;

      case 'dev':
      case 'development':
        if (isNpmPackage()) {
          log.error('Development mode is not available for npm package installations.');
          log.info('Please clone the repository from GitHub to use development mode.');
          process.exit(1);
        }

        log.info('Starting ccdash in development mode...');
        const devBackendUrl = `http://localhost:6060`;
        const devFrontendUrl = frontendUrl || `http://localhost:${frontendPort}`;
        const frontendTargetUrl = frontendUrl || `http://localhost:${frontendPort}`;
        const apiUrl = `http://localhost:6060`;
        
        // Add debugging info
        log.info(`Backend will bind to all interfaces on port 6060`);
        log.info(`Frontend will connect to: ${apiUrl}`);
        
        log.info(`Backend:  ${devBackendUrl}`);
        log.info(`Frontend: ${devFrontendUrl}`);

        // Backend host is fixed to localhost
        const devBackendHost = 'localhost';

        log.info(`Setting backend FRONTEND_URL to: ${frontendTargetUrl}`);
        log.info(`Backend will bind to: ${devBackendHost}:6060`);
        
        // Show warnings for disabled security features
        if (disableSafetyCheck) {
          log.warning('⚠️  Command safety checks are DISABLED - all commands will execute without validation!');
        }
        if (disableAuth) {
          log.warning('⚠️  API authentication is DISABLED - no API key required!');
        }
        
        const devBackendEnv = {
          ...process.env,
          PORT: '6060',
          HOST: devBackendHost,
          FRONTEND_URL: frontendTargetUrl
        };
        
        // Apply CLI overrides
        if (disableSafetyCheck) {
          devBackendEnv.COMMAND_WHITELIST_ENABLED = 'false';
        }
        if (disableAuth) {
          devBackendEnv.CCDASH_API_KEY = ''; // Clear API key to disable auth
          devBackendEnv.GIN_MODE = 'debug'; // Force debug mode to disable auth
        } else if (apiKey) {
          // Set API key from command line
          devBackendEnv.CCDASH_API_KEY = apiKey;
          log.info(`🔑 Using API key from command line`);
        }
        
        const devBackend = spawn('go', ['run', 'cmd/server/main.go'], {
          cwd: backendPath,
          stdio: 'inherit',
          env: devBackendEnv
        });

        // Extract hostname from frontend URL for binding
        let hostname = 'localhost';
        if (frontendUrl) {
          try {
            const url = new URL(frontendUrl);
            hostname = url.hostname;
          } catch (error) {
            log.warning(`Invalid frontend URL: ${frontendUrl}. Using localhost`);
          }
        }

        log.info(`Setting frontend NEXT_PUBLIC_API_URL to: ${apiUrl}/api`);
        log.info(`Frontend will bind to: ${hostname}:${frontendPort}`);
        const devFrontend = spawn('npm', ['run', 'dev'], {
          cwd: frontendPath,
          stdio: 'inherit',
          env: {
            ...process.env,
            PORT: frontendPort.toString(),
            HOSTNAME: hostname,
            NEXT_PUBLIC_API_URL: `${apiUrl}/api`
          }
        });

        const devCleanup = () => {
          log.info('Stopping development servers...');
          if (devBackend && !devBackend.killed) {
            devBackend.kill('SIGTERM');
          }
          if (devFrontend && !devFrontend.killed) {
            devFrontend.kill('SIGTERM');
          }
          process.exit(0);
        };

        process.on('SIGINT', devCleanup);
        process.on('SIGTERM', devCleanup);

        // Open browser for development mode if requested
        if (shouldOpenBrowser) {
          log.info('Development mode - waiting for services to be ready...');
          
          (async () => {
            try {
              // Wait for both services to be ready
              await Promise.all([
                waitForService(devBackendUrl + '/api/health'),
                waitForService(devFrontendUrl)
              ]);
              
              // Open browser after services are ready
              setTimeout(() => {
                openBrowser(devFrontendUrl);
              }, 2000); // Longer delay for dev mode as it takes more time to start
              
            } catch (error) {
              log.warning('Development services took too long to start, please open browser manually');
              log.info(`Frontend: ${devFrontendUrl}`);
            }
          })();
        }

        break;

      case 'help':
      case '--help':
      case '-h':
        console.log(`
Usage: ccdash [command] [options]

Commands:
  start, run    Start ccdash (default)
  dev           Start in development mode
  build         Build the application
  help          Show this help message

Options:
  --frontend-port, -fp        Frontend server port (default: 3000)
  --frontend-url, -fu         Frontend server URL (overrides frontend-port)
  --no-open, --no-browser     Don't open browser automatically
  --open, --browser           Open browser automatically (default)
  --no-safety, --disable-safety-check    Disable command safety checks (⚠️ DANGEROUS)
  --no-auth, --disable-auth              Disable API authentication
  --api-key, -k               Specify API key for authentication

Note: Backend port is fixed at 6060 for npm package distribution

Examples:
  npx ccdash                                              # Start with default ports and open browser
  npx ccdash --no-open                                    # Start without opening browser
  npx ccdash --frontend-port 3001                         # Start with custom frontend port
  npx ccdash -fp 3001                                     # Start with custom frontend port (short form)
  npx ccdash --frontend-url https://app.example.com       # Use custom frontend URL
  npx ccdash --api-key abc123xyz                          # Use specific API key
  npx ccdash --api-key=abc123xyz                          # Use specific API key (alternative syntax)
  npx ccdash -k abc123xyz                                 # Use specific API key (short form)
  npx ccdash --no-safety                                  # Start without command safety checks (⚠️ DANGEROUS)
  npx ccdash --no-auth                                    # Start without API authentication
  npx ccdash --no-safety --no-auth                        # Disable all security features (⚠️ VERY DANGEROUS)
  npx ccdash dev --frontend-port 3001 --no-browser        # Development mode without browser
  npx ccdash build                                        # Build the application

API Key Options:
  1. Command line: --api-key your-key-here
  2. Environment variable: CCDASH_API_KEY=your-key-here
  3. .env file: Add CCDASH_API_KEY=your-key-here
  4. Auto-generated: CCDash will generate one automatically if none provided

Browser Launch:
  By default, ccdash will automatically open your browser when services are ready.
  Use --no-open to disable automatic browser launch.

For more information, visit: https://github.com/satoshi03/ccdash
        `);
        break;

      case 'version':
      case '--version':
      case '-v':
        console.log('ccdash v1.0.0');
        break;

      default:
        log.error(`Unknown command: ${command}`);
        log.info('Run "ccdash help" for available commands');
        process.exit(1);
    }

  } catch (error) {
    log.error(`Error: ${error.message}`);
    process.exit(1);
  }
}

// Handle uncaught exceptions
process.on('uncaughtException', (error) => {
  log.error(`Uncaught Exception: ${error.message}`);
  process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
  log.error(`Unhandled Rejection at: ${promise}, reason: ${reason}`);
  process.exit(1);
});

// Run the CLI
main();
