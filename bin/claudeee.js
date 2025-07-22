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
const frontendPath = path.join(packageRoot, 'frontend');

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
    
    const installProcess = spawn('npm', ['install'], {
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

// Start backend server
function startBackend() {
  const serverPath = path.join(packageRoot, 'bin', 'claudeee-server');
  
  if (!fs.existsSync(serverPath)) {
    log.error('Backend server not found. Please run build first.');
    process.exit(1);
  }
  
  log.info('Starting backend server on http://localhost:8080');
  const backendProcess = spawn(serverPath, [], {
    stdio: 'inherit',
    detached: false
  });
  
  return backendProcess;
}

// Start frontend server
function startFrontend() {
  log.info('Starting frontend server on http://localhost:3000');
  const frontendProcess = spawn('npm', ['run', 'start'], {
    cwd: frontendPath,
    stdio: 'inherit',
    detached: false
  });
  
  return frontendProcess;
}

// Main CLI function
async function main() {
  const args = process.argv.slice(2);
  const command = args[0] || 'start';
  
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
        
        // Install frontend dependencies if needed
        if (!checkNodeDependencies()) {
          await installFrontendDependencies();
        }
        
        // Start both services
        const backendProcess = startBackend();
        const frontendProcess = startFrontend();
        
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
        log.info('Backend:  http://localhost:8080');
        log.info('Frontend: http://localhost:3000');
        log.info('Press Ctrl+C to stop');
        
        break;
        
      case 'build':
        log.info('Building claudeee...');
        await buildBackend();
        if (!checkNodeDependencies()) {
          await installFrontendDependencies();
        }
        await buildFrontend();
        log.success('Build completed successfully!');
        break;
        
      case 'dev':
      case 'development':
        log.info('Starting claudeee in development mode...');
        
        const devBackend = spawn('go', ['run', 'cmd/server/main.go'], {
          cwd: backendPath,
          stdio: 'inherit'
        });
        
        const devFrontend = spawn('npm', ['run', 'dev'], {
          cwd: frontendPath,
          stdio: 'inherit'
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
Usage: claudeee [command]

Commands:
  start, run    Start claudeee (default)
  dev           Start in development mode
  build         Build the application
  help          Show this help message

Examples:
  npx claudeee           # Start the application
  npx claudeee dev       # Start in development mode
  npx claudeee build     # Build the application
  npx claudeee help      # Show help

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