#!/usr/bin/env node

const { spawn } = require('child_process');
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
  let backendPort = 8080;
  let frontendPort = 3000;
  let backendUrl = null;
  let frontendUrl = null;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (arg === '--backend-port' || arg === '-bp') {
      backendPort = parseInt(args[i + 1]) || 8080;
      i++;
    } else if (arg === '--frontend-port' || arg === '-fp') {
      frontendPort = parseInt(args[i + 1]) || 3000;
      i++;
    } else if (arg === '--backend-url' || arg === '-bu') {
      backendUrl = args[i + 1];
      i++;
    } else if (arg === '--frontend-url' || arg === '-fu') {
      frontendUrl = args[i + 1];
      i++;
    } else if (!arg.startsWith('-')) {
      command = arg;
    }
  }

  // Extract port from URL if provided
  if (backendUrl) {
    try {
      const url = new URL(backendUrl);
      backendPort = parseInt(url.port) || (url.protocol === 'https:' ? 443 : 80);
    } catch (error) {
      log.warning(`Invalid backend URL: ${backendUrl}. Using default port ${backendPort}`);
    }
  }

  if (frontendUrl) {
    try {
      const url = new URL(frontendUrl);
      frontendPort = parseInt(url.port) || (url.protocol === 'https:' ? 443 : 80);
    } catch (error) {
      log.warning(`Invalid frontend URL: ${frontendUrl}. Using default port ${frontendPort}`);
    }
  }

  return { command, backendPort, frontendPort, backendUrl, frontendUrl };
}

// Start backend server with custom port or URL
function startBackend(port = 8080, frontendPort = 3000, frontendUrl = null, backendUrl = null) {
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
  log.info(`Starting backend server on ${backendDisplayUrl}`);
  
  const backendProcess = spawn(serverPath, [], {
    stdio: 'inherit',
    detached: false,
    env: {
      ...process.env,
      PORT: port.toString(),
      HOST: backendHost,
      FRONTEND_URL: frontendTargetUrl
    }
  });

  return backendProcess;
}

// Start frontend server with custom port or URL
function startFrontend(port = 3000, backendPort = 8080, backendUrl = null, frontendUrl = null) {
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
    const backendTargetUrl = backendUrl || `http://localhost:${backendPort}`;
    log.info(`API server is running on ${backendTargetUrl}`);
    return null;
  }

  const nextDir = path.join(frontendPath, '.next');
  const standalonePath = path.join(frontendPath, '.next', 'standalone', 'server.js');
  const apiUrl = backendUrl || `http://localhost:${backendPort}`;

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
          NEXT_PUBLIC_API_URL: `${apiUrl}/api`,
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
          NEXT_PUBLIC_API_URL: `${apiUrl}/api`,
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
          NEXT_PUBLIC_API_URL: `${apiUrl}/api`,
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
  const { command, backendPort, frontendPort, backendUrl, frontendUrl } = parseArgs();

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

          // Build frontend if standalone build doesn't exist or if custom backend URL is provided
          const standalonePath = path.join(frontendPath, '.next', 'standalone', 'server.js');
          const needsRebuild = !fs.existsSync(standalonePath) || backendUrl;
          
          if (needsRebuild) {
            if (backendUrl) {
              log.info(`Custom backend URL provided: ${backendUrl}. Rebuilding frontend...`);
            }
            await buildFrontendWithApiUrl(backendUrl ? `${backendUrl}/api` : null);
          }
        } else {
          log.info('Running from npm package, skipping frontend build');
        }

        // Start both services
        const backendProcess = startBackend(backendPort, frontendPort, frontendUrl, backendUrl);
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
        const backendHost = backendUrl ? new URL(backendUrl).hostname : 'localhost';
        const backendDisplayUrl = backendUrl || `http://${backendHost}:${backendPort}`;
        const frontendDisplayUrl = frontendUrl || `http://localhost:${frontendPort}`;
        log.info(`Backend:  ${backendDisplayUrl}`);
        if (frontendProcess) {
          log.info(`Frontend: ${frontendDisplayUrl}`);
        }
        log.info('Press Ctrl+C to stop');

        break;

      case 'build':
        log.info('Building ccdash...');
        await buildBackend();

        if (!isNpmPackage()) {
          if (!checkNodeDependencies()) {
            await installFrontendDependencies();
          }
          await buildFrontendWithApiUrl(backendUrl ? `${backendUrl}/api` : null);
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
        const devBackendUrl = backendUrl || `http://localhost:${backendPort}`;
        const devFrontendUrl = frontendUrl || `http://localhost:${frontendPort}`;
        const frontendTargetUrl = frontendUrl || `http://localhost:${frontendPort}`;
        const apiUrl = backendUrl || `http://localhost:${backendPort}`;
        
        // Add debugging info
        log.info(`Backend will bind to all interfaces on port ${backendPort}`);
        log.info(`Frontend will connect to: ${apiUrl}`);
        
        log.info(`Backend:  ${devBackendUrl}`);
        log.info(`Frontend: ${devFrontendUrl}`);

        // Extract host from backend URL if provided
        let devBackendHost = 'localhost';
        if (backendUrl) {
          try {
            const url = new URL(backendUrl);
            devBackendHost = url.hostname;
          } catch (error) {
            log.warning(`Invalid backend URL: ${backendUrl}. Using localhost`);
          }
        }

        log.info(`Setting backend FRONTEND_URL to: ${frontendTargetUrl}`);
        log.info(`Backend will bind to: ${devBackendHost}:${backendPort}`);
        const devBackend = spawn('go', ['run', 'cmd/server/main.go'], {
          cwd: backendPath,
          stdio: 'inherit',
          env: {
            ...process.env,
            PORT: backendPort.toString(),
            HOST: devBackendHost,
            FRONTEND_URL: frontendTargetUrl
          }
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
  --backend-port, -bp    Backend server port (default: 8080)
  --frontend-port, -fp   Frontend server port (default: 3000)
  --backend-url, -bu     Backend server URL (overrides backend-port)
  --frontend-url, -fu    Frontend server URL (overrides frontend-port)

Examples:
  npx ccdash                                              # Start with default ports
  npx ccdash --backend-port 8081                          # Start with custom backend port
  npx ccdash -bp 8081 -fp 3001                            # Start with custom ports
  npx ccdash --backend-url http://api.example.com         # Use custom backend URL
  npx ccdash --frontend-url https://app.example.com       # Use custom frontend URL
  npx ccdash -bu http://localhost:8081 -fu http://localhost:3001  # Custom URLs
  npx ccdash dev --backend-port 8081                      # Development mode with custom backend port
  npx ccdash build                                        # Build the application

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
