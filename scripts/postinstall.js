#!/usr/bin/env node

const path = require('path');
const fs = require('fs');
const PlatformDetector = require('./platform-detector');

/**
 * Post-install script for claudeee npm package
 * Sets up the appropriate binary for the current platform
 */

class PostInstallSetup {
  constructor() {
    this.detector = new PlatformDetector();
    this.packageRoot = path.resolve(__dirname, '..');
    this.binDir = path.join(this.packageRoot, 'bin');
    this.targetBinary = path.join(this.binDir, 'claudeee-server');
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
   * Main setup process
   */
  async setup() {
    try {
      this.log('Starting claudeee post-install setup...');
      
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

      this.log('claudeee setup completed successfully!', 'success');
      this.log(`Binary location: ${this.targetBinary}`, 'info');
      this.log('You can now run: claudeee --help', 'info');

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
      if (packageJson.bin && packageJson.bin.claudeee) {
        const expectedBinPath = './bin/claudeee-server';
        if (packageJson.bin.claudeee !== expectedBinPath) {
          packageJson.bin.claudeee = expectedBinPath;
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
      this.log('Running claudeee cleanup...');
      
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
      setup.validateBinary(process.argv[3] || path.join(__dirname, '..', 'bin', 'claudeee-server'))
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