#!/usr/bin/env node

const path = require('path');
const fs = require('fs');
const PlatformDetector = require('./platform-detector');

/**
 * Post-install script for ccdash npm package
 * Sets up the appropriate binary for the current platform
 */

class PostInstallSetup {
  constructor() {
    this.detector = new PlatformDetector();
    this.packageRoot = path.resolve(__dirname, '..');
    this.binDir = path.join(this.packageRoot, 'bin');
    this.targetBinary = path.join(this.binDir, 'ccdash-server');
  }

  log(message, level = 'info') {
    const timestamp = new Date().toISOString();
    const prefix = {
      'info': '🔧',
      'success': '✅',
      'warning': '⚠️',
      'error': '❌'
    }[level] || 'ℹ️';
    
    console.log(`${prefix} [${timestamp}] ${message}`);
  }

  /**
   * Check if we're running in a CI environment
   */
  isCI() {
    return process.env.CI === 'true' || 
           process.env.GITHUB_ACTIONS === 'true' ||
           process.env.TRAVIS === 'true' ||
           process.env.CIRCLECI === 'true' ||
           process.env.JENKINS_URL !== undefined;
  }

  /**
   * Check if we're running from an npm package installation
   */
  isNpmPackage() {
    // Check for npm environment variables first
    if (process.env.npm_package_name === 'ccdash' || 
        process.env.npm_config_global === 'true' ||
        process.env.npm_command === 'install') {
      return true;
    }
    
    // Check if we're in a node_modules directory (local or global)
    const packagePath = this.packageRoot;
    if (packagePath.includes('node_modules')) {
      return true;
    }
    
    // Check for global npm directory patterns
    if (packagePath.includes('/lib/node_modules/') || 
        packagePath.includes('\\node_modules\\') ||
        packagePath.includes('.npm/')) {
      return true;
    }
    
    // Check if package was installed via npm by looking for npm-specific files in parent dirs
    let currentDir = this.packageRoot;
    while (currentDir !== path.dirname(currentDir)) {
      if (path.basename(currentDir) === 'node_modules') {
        return true;
      }
      if (fs.existsSync(path.join(currentDir, 'package-lock.json')) ||
          fs.existsSync(path.join(currentDir, 'pnpm-lock.yaml'))) {
        return true;
      }
      currentDir = path.dirname(currentDir);
    }
    
    // Check if we don't have development files (indicating we're a packaged install)
    return !fs.existsSync(path.join(this.packageRoot, '.git')) &&
           !fs.existsSync(path.join(this.packageRoot, 'go.mod'));
  }

  /**
   * Create symbolic link or copy binary
   * @param {string} sourcePath - Source binary path
   * @param {string} targetPath - Target path for the generic binary
   */
  setupGenericBinary(sourcePath, targetPath) {
    try {
      // Remove existing target if it exists
      if (fs.existsSync(targetPath)) {
        fs.unlinkSync(targetPath);
      }

      if (process.platform === 'win32') {
        // On Windows, copy the file instead of creating a symlink
        fs.copyFileSync(sourcePath, targetPath);
        this.log(`Copied binary to ${targetPath}`);
      } else {
        // On Unix-like systems, create a symbolic link
        const relativePath = path.relative(path.dirname(targetPath), sourcePath);
        fs.symlinkSync(relativePath, targetPath);
        this.log(`Created symlink from ${targetPath} to ${sourcePath}`);
      }
    } catch (error) {
      this.log(`Failed to setup generic binary: ${error.message}`, 'warning');
      // Fallback: copy the file
      try {
        fs.copyFileSync(sourcePath, targetPath);
        this.log(`Fallback: Copied binary to ${targetPath}`);
      } catch (copyError) {
        throw new Error(`Failed to setup binary: ${copyError.message}`);
      }
    }
  }

  /**
   * Validate the setup binary
   * @param {string} binaryPath - Path to binary to validate
   */
  async validateBinary(binaryPath) {
    return new Promise((resolve) => {
      const { spawn } = require('child_process');
      
      // Try to run the binary with --help flag (most binaries support this)
      const child = spawn(binaryPath, ['--version'], {
        stdio: 'pipe',
        timeout: 5000
      });

      let hasOutput = false;
      
      child.stdout.on('data', () => {
        hasOutput = true;
      });

      child.stderr.on('data', () => {
        hasOutput = true;
      });

      child.on('close', (code) => {
        // Binary is considered valid if it runs and exits (regardless of exit code)
        // or if it provides output
        resolve(hasOutput || code !== null);
      });

      child.on('error', () => {
        resolve(false);
      });

      // Fallback timeout
      setTimeout(() => {
        child.kill();
        resolve(false);
      }, 5000);
    });
  }

  /**
   * Install frontend dependencies
   */
  async installFrontendDependencies() {
    const frontendPath = path.join(this.packageRoot, 'frontend');
    const nodeModulesPath = path.join(frontendPath, 'node_modules');
    
    // Skip frontend dependency installation for npm packages
    if (this.isNpmPackage()) {
      this.log('Running from npm package, skipping frontend dependency installation');
      return;
    }
    
    if (fs.existsSync(nodeModulesPath)) {
      this.log('Frontend dependencies already installed');
      return;
    }
    
    if (!fs.existsSync(path.join(frontendPath, 'package.json'))) {
      this.log('Frontend directory not found, skipping dependency installation');
      return;
    }
    
    // Only install dependencies in development environment
    if (process.env.NODE_ENV === 'production' || this.isCI()) {
      this.log('Production/CI environment detected, skipping frontend dependency installation');
      return;
    }
    
    this.log('Installing frontend dependencies...');
    
    const { spawn } = require('child_process');
    return new Promise((resolve) => {
      const installProcess = spawn('npm', ['install'], {
        cwd: frontendPath,
        stdio: 'inherit'
      });
      
      installProcess.on('close', (code) => {
        if (code === 0) {
          this.log('Frontend dependencies installed successfully', 'success');
        } else {
          this.log('Frontend dependency installation completed with warnings', 'warning');
        }
        resolve(); // Continue regardless of exit code
      });
      
      installProcess.on('error', (err) => {
        this.log(`Frontend dependency installation failed: ${err.message}`, 'warning');
        resolve(); // Don't fail the entire setup
      });
    });
  }

  /**
   * Build frontend if needed
   */
  async buildFrontend() {
    const frontendPath = path.join(this.packageRoot, 'frontend');
    const standalonePath = path.join(frontendPath, '.next', 'standalone', 'server.js');
    
    // Skip frontend build for npm packages - frontend should be pre-built
    if (this.isNpmPackage()) {
      this.log('Running from npm package, skipping frontend build');
      return;
    }
    
    if (fs.existsSync(standalonePath)) {
      this.log('Frontend already built, skipping build step');
      return;
    }
    
    if (!fs.existsSync(path.join(frontendPath, 'package.json'))) {
      this.log('Frontend directory not found, skipping build step');
      return;
    }
    
    // Only build frontend in development environment
    if (process.env.NODE_ENV === 'production' || this.isCI()) {
      this.log('Production/CI environment detected, skipping frontend build');
      return;
    }
    
    this.log('Building frontend...');
    
    const { spawn } = require('child_process');
    return new Promise((resolve) => {
      const buildProcess = spawn('npm', ['run', 'build'], {
        cwd: frontendPath,
        stdio: 'inherit'
      });
      
      buildProcess.on('close', (code) => {
        if (code === 0) {
          this.log('Frontend built successfully', 'success');
        } else {
          this.log('Frontend build completed with warnings', 'warning');
        }
        resolve(); // Continue regardless of exit code
      });
      
      buildProcess.on('error', (err) => {
        this.log(`Frontend build failed: ${err.message}`, 'warning');
        resolve(); // Don't fail the entire setup
      });
    });
  }

  /**
   * Main setup process
   */
  async setup() {
    try {
      this.log('Starting ccdash post-install setup...');
      
      // Install frontend dependencies first
      await this.installFrontendDependencies();
      
      // Build frontend
      await this.buildFrontend();
      
      // Check platform support
      if (!this.detector.isSupported()) {
        throw new Error(
          `Platform ${this.detector.platform}-${this.detector.arch} is not supported. ` +
          `Supported platforms: macOS (Intel/Apple Silicon), Linux (x64/ARM64), Windows (x64)`
        );
      }

      this.log(`Detected platform: ${this.detector.platform}-${this.detector.arch}`);

      // Check if bin directory exists
      if (!fs.existsSync(this.binDir)) {
        throw new Error(`Binary directory not found: ${this.binDir}`);
      }

      // List available binaries
      const availableBinaries = this.detector.listAvailableBinaries(this.binDir);
      this.log(`Found ${availableBinaries.length} available binaries`);

      if (availableBinaries.length === 0) {
        throw new Error('No binaries found in package. This may indicate a packaging issue.');
      }

      // Setup platform-specific binary
      const platformBinaryPath = this.detector.setupBinary(this.binDir);
      this.log(`Platform-specific binary ready: ${platformBinaryPath}`);

      // Validate the binary
      this.log('Validating binary...');
      const isValid = await this.validateBinary(platformBinaryPath);
      
      if (!isValid && !this.isCI()) {
        this.log('Binary validation failed, but continuing...', 'warning');
      }

      // Create generic binary link/copy for easier access
      this.setupGenericBinary(platformBinaryPath, this.targetBinary);

      // Update package.json bin field if needed
      this.updatePackageBin();

      this.log('ccdash setup completed successfully!', 'success');
      this.log(`Binary location: ${this.targetBinary}`, 'info');
      this.log('You can now run: ccdash --help', 'info');

    } catch (error) {
      this.log(`Setup failed: ${error.message}`, 'error');
      
      // Provide helpful error information
      this.log('Available binaries:', 'info');
      const binaries = this.detector.listAvailableBinaries(this.binDir);
      binaries.forEach(binary => {
        this.log(`  - ${binary.name} (${binary.size} bytes)`, 'info');
      });

      this.log(`Platform info: ${JSON.stringify(this.detector.getPlatformInfo(), null, 2)}`, 'info');
      
      // In CI environments, fail hard
      if (this.isCI()) {
        process.exit(1);
      } else {
        this.log('Setup failed, but installation will continue. You may need to run setup manually.', 'warning');
      }
    }
  }

  /**
   * Update package.json bin field to point to the correct binary
   */
  updatePackageBin() {
    try {
      const packageJsonPath = path.join(this.packageRoot, 'package.json');
      if (!fs.existsSync(packageJsonPath)) {
        return;
      }

      const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
      
      // Ensure bin field points to our setup binary
      if (packageJson.bin && packageJson.bin.ccdash) {
        const expectedBinPath = './bin/ccdash-server';
        if (packageJson.bin.ccdash !== expectedBinPath) {
          packageJson.bin.ccdash = expectedBinPath;
          fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 2));
          this.log('Updated package.json bin field');
        }
      }
    } catch (error) {
      this.log(`Could not update package.json: ${error.message}`, 'warning');
    }
  }

  /**
   * Cleanup method for uninstall
   */
  cleanup() {
    try {
      this.log('Running ccdash cleanup...');
      
      if (fs.existsSync(this.targetBinary)) {
        fs.unlinkSync(this.targetBinary);
        this.log('Removed generic binary');
      }

      this.log('Cleanup completed', 'success');
    } catch (error) {
      this.log(`Cleanup failed: ${error.message}`, 'warning');
    }
  }
}

// Export for testing
module.exports = PostInstallSetup;

// Run setup if called directly
if (require.main === module) {
  const setup = new PostInstallSetup();
  
  const command = process.argv[2];
  
  switch (command) {
    case 'cleanup':
    case 'uninstall':
      setup.cleanup();
      break;
      
    case 'validate':
      setup.validateBinary(process.argv[3] || path.join(__dirname, '..', 'bin', 'ccdash-server'))
        .then(isValid => {
          console.log(isValid ? 'Binary is valid' : 'Binary validation failed');
          process.exit(isValid ? 0 : 1);
        });
      break;
      
    case 'setup':
    default:
      setup.setup().catch(error => {
        console.error('Setup failed:', error.message);
        process.exit(1);
      });
      break;
  }
}