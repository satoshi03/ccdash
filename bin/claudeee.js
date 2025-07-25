#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

// ASCII Art Logo
const LOGO = `
 ██████╗██╗      █████╗ ██╗   ██╗██████╗ ███████╗███████╗███████╗
██╔════╝██║     ██╔══██╗██║   ██║██╔══██╗██╔════╝██╔════╝██╔════╝
██║     ██║     ███████║██║   ██║██║  ██║█████╗  █████╗  █████╗  
██║     ██║     ██╔══██║██║   ██║██║  ██║██╔══╝  ██╔══╝  ██╔══╝  
╚██████╗███████╗██║  ██║╚██████╔╝██████╔╝███████╗███████╗███████╗
 ╚═════╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝╚══════╝╚══════╝
                                                                  
        Claude Code Monitoring & Task Scheduler v1.0.0
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
const frontendPath = path.join(packageRoot, 'bin', 'frontend-dist');

// Check if we're running from an npm package installation
function isNpmPackage() {
  // Check for npm environment variables
  if (process.env.npm_package_name === 'claudeee' || 
      process.env.npm_config_global === 'true') {
    return true;
  }
  
  // Check if we're in a node_modules directory
  if (packageRoot.includes('node_modules')) {
    return true;
  }
  
  // Check if we don't have development files (indicating we're a packaged install)
  return !fs.existsSync(path.join(packageRoot, '.git')) &&
         !fs.existsSync(path.join(packageRoot, 'go.mod'));
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
    
    const buildProcess = spawn('go', ['build', '-o', path.join(packageRoot, 'bin', 'claudeee-server'), 'cmd/server/main.go'], {
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
  return new Promise((resolve, reject) => {
    log.info('Building frontend...');
    
    const buildProcess = spawn('npm', ['run', 'build'], {
      cwd: frontendPath,
      stdio: 'inherit'
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
  
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    
    if (arg === '--backend-port' || arg === '-bp') {
      backendPort = parseInt(args[i + 1]) || 8080;
      i++;
    } else if (arg === '--frontend-port' || arg === '-fp') {
      frontendPort = parseInt(args[i + 1]) || 3000;
      i++;
    } else if (!arg.startsWith('-')) {
      command = arg;
    }
  }
  
  return { command, backendPort, frontendPort };
}

// Start backend server with custom port
function startBackend(port = 8080, frontendPort = 3000) {
  const serverPath = path.join(packageRoot, 'bin', 'claudeee-server');
  
  if (!fs.existsSync(serverPath)) {
    log.error('Backend server not found. Please run build first.');
    process.exit(1);
  }
  
  log.info(`Starting backend server on http://localhost:${port}`);
  const backendProcess = spawn(serverPath, [], {
    stdio: 'inherit',
    detached: false,
    env: {
      ...process.env,
      PORT: port.toString(),
      FRONTEND_URL: `http://localhost:${frontendPort}`
    }
  });
  
  return backendProcess;
}

// Start frontend server with custom port
function startFrontend(port = 3000, backendPort = 8080) {
  // For npm packages, check if frontend-dist exists
  if (!fs.existsSync(frontendPath)) {
    log.warning('Frontend not available in this installation');
    log.info(`API server is running on http://localhost:${backendPort}`);
    return null;
  }
  
  // Check if standalone build exists
  const standalonePath = path.join(frontendPath, '.next', 'standalone', 'server.js');
  const nextDir = path.join(frontendPath, '.next');
  
  // For npm packages (bin/frontend-dist structure), try Next.js start directly
  if (isNpmPackage() && fs.existsSync(nextDir)) {
    log.info(`Starting frontend server on http://localhost:${port}`);
    log.info('Using pre-built Next.js frontend');
    
    const frontendProcess = spawn('npm', ['start'], {
      cwd: frontendPath,
      stdio: 'inherit',
      detached: false,
      env: {
        ...process.env,
        PORT: port.toString(),
        NEXT_PUBLIC_API_URL: `http://localhost:${backendPort}`
      }
    });
    
    return frontendProcess;
  }
  
  // For development environments (original frontend/ structure)
  if (!isNpmPackage()) {
    log.info(`Starting frontend server on http://localhost:${port}`);
    
    if (fs.existsSync(standalonePath)) {
      // Use standalone build
      log.info('Using standalone Next.js build');
      const standaloneDir = path.join(frontendPath, '.next', 'standalone');
      const frontendProcess = spawn('node', ['server.js'], {
        cwd: standaloneDir,
        stdio: 'inherit',
        detached: false,
        env: {
          ...process.env,
          PORT: port.toString(),
          NEXT_PUBLIC_API_URL: `http://localhost:${backendPort}`,
          HOSTNAME: '0.0.0.0'
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
          NEXT_PUBLIC_API_URL: `http://localhost:${backendPort}`
        }
      });
      return frontendProcess;
    }
  }
  
  log.warning('Frontend not available');
  return null;
}

// Main CLI function
async function main() {
  const { command, backendPort, frontendPort } = parseArgs();
  
  log.logo();
  
  try {
    switch (command) {
      case 'start':
      case 'run':
        log.info('Starting claudeee...');
        
        // Check prerequisites
        const hasGo = await checkGoInstallation();
        if (!hasGo) {
          log.error('Go is not installed. Please install Go 1.21 or later.');
          log.info('Visit https://golang.org/dl/ to download Go');
          process.exit(1);
        }
        
        // Build backend if needed
        const serverPath = path.join(packageRoot, 'bin', 'claudeee-server');
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
          if (!fs.existsSync(standalonePath)) {
            await buildFrontend();
          }
        } else {
          log.info('Running from npm package, skipping frontend build');
        }
        
        // Start both services
        const backendProcess = startBackend(backendPort, frontendPort);
        const frontendProcess = startFrontend(frontendPort, backendPort);
        
        // Handle process cleanup
        const cleanup = () => {
          log.info('Shutting down claudeee...');
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
        
        log.success('claudeee is running!');
        log.info(`Backend:  http://localhost:${backendPort}`);
        if (frontendProcess) {
          log.info(`Frontend: http://localhost:${frontendPort}`);
        }
        log.info('Press Ctrl+C to stop');
        
        break;
        
      case 'build':
        log.info('Building claudeee...');
        await buildBackend();
        
        if (!isNpmPackage()) {
          if (!checkNodeDependencies()) {
            await installFrontendDependencies();
          }
          await buildFrontend();
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
        
        log.info('Starting claudeee in development mode...');
        log.info(`Backend:  http://localhost:${backendPort}`);
        log.info(`Frontend: http://localhost:${frontendPort}`);
        
        const devBackend = spawn('go', ['run', 'cmd/server/main.go'], {
          cwd: backendPath,
          stdio: 'inherit',
          env: {
            ...process.env,
            PORT: backendPort.toString(),
            FRONTEND_URL: `http://localhost:${frontendPort}`
          }
        });
        
        const devFrontend = spawn('npm', ['run', 'dev'], {
          cwd: frontendPath,
          stdio: 'inherit',
          env: {
            ...process.env,
            PORT: frontendPort.toString(),
            NEXT_PUBLIC_API_URL: `http://localhost:${backendPort}`
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
Usage: claudeee [command] [options]

Commands:
  start, run    Start claudeee (default)
  dev           Start in development mode
  build         Build the application
  help          Show this help message

Options:
  --backend-port, -bp   Backend server port (default: 8080)
  --frontend-port, -fp  Frontend server port (default: 3000)

Examples:
  npx claudeee                           # Start with default ports
  npx claudeee --backend-port 8081       # Start with custom backend port
  npx claudeee -bp 8081 -fp 3001         # Start with custom ports
  npx claudeee dev --backend-port 8081   # Development mode with custom backend port
  npx claudeee build                     # Build the application

For more information, visit: https://github.com/claudeee/claudeee
        `);
        break;
        
      case 'version':
      case '--version':
      case '-v':
        console.log('claudeee v1.0.0');
        break;
        
      default:
        log.error(`Unknown command: ${command}`);
        log.info('Run "claudeee help" for available commands');
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