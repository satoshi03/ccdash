#!/usr/bin/env node

const os = require('os');
const path = require('path');
const fs = require('fs');

/**
 * Platform Detection and Binary Management
 * Detects the current platform and returns the appropriate binary name
 */

class PlatformDetector {
  constructor() {
    this.platform = process.platform;
    this.arch = process.arch;
  }

  /**
   * Get the binary suffix for the current platform
   * @returns {string} Binary suffix (e.g., 'darwin-amd64', 'windows-amd64.exe')
   */
  getBinarySuffix() {
    const platformMap = {
      'darwin': {
        'x64': 'darwin-amd64',
        'arm64': 'darwin-arm64'
      },
      'linux': {
        'x64': 'linux-amd64'
      },
      'win32': {
        'x64': 'windows-amd64.exe'
      }
    };

    const platform = platformMap[this.platform];
    if (!platform) {
      throw new Error(`Unsupported platform: ${this.platform}`);
    }

    const archMapping = {
      'x64': 'x64',
      'arm64': 'arm64'
    };

    const normalizedArch = archMapping[this.arch];
    if (!normalizedArch) {
      throw new Error(`Unsupported architecture: ${this.arch} on ${this.platform}`);
    }

    const suffix = platform[normalizedArch];
    if (!suffix) {
      throw new Error(`Unsupported architecture ${this.arch} for platform ${this.platform}`);
    }

    return suffix;
  }

  /**
   * Get the full binary name for the current platform
   * @returns {string} Full binary name
   */
  getBinaryName() {
    return `ccdash-server-${this.getBinarySuffix()}`;
  }

  /**
   * Get the path to the binary for the current platform
   * @param {string} binDir - Directory containing binaries
   * @returns {string} Full path to binary
   */
  getBinaryPath(binDir) {
    const binaryName = this.getBinaryName();
    return path.join(binDir, binaryName);
  }

  /**
   * Check if the binary exists for the current platform
   * @param {string} binDir - Directory containing binaries
   * @returns {boolean} True if binary exists
   */
  binaryExists(binDir) {
    const binaryPath = this.getBinaryPath(binDir);
    return fs.existsSync(binaryPath);
  }

  /**
   * Get platform information
   * @returns {object} Platform information
   */
  getPlatformInfo() {
    return {
      platform: this.platform,
      arch: this.arch,
      binarySuffix: this.getBinarySuffix(),
      binaryName: this.getBinaryName(),
      supported: this.isSupported()
    };
  }

  /**
   * Check if the current platform is supported
   * @returns {boolean} True if platform is supported
   */
  isSupported() {
    try {
      this.getBinarySuffix();
      return true;
    } catch (error) {
      return false;
    }
  }

  /**
   * List all available binaries in the bin directory
   * @param {string} binDir - Directory containing binaries
   * @returns {Array} List of available binaries
   */
  listAvailableBinaries(binDir) {
    if (!fs.existsSync(binDir)) {
      return [];
    }

    return fs.readdirSync(binDir)
      .filter(file => file.startsWith('ccdash-server-'))
      .map(file => {
        const fullPath = path.join(binDir, file);
        const stats = fs.statSync(fullPath);
        return {
          name: file,
          path: fullPath,
          size: stats.size,
          executable: !!(stats.mode & parseInt('111', 8))
        };
      });
  }

  /**
   * Make binary executable (Unix-like systems)
   * @param {string} binaryPath - Path to binary
   */
  makeExecutable(binaryPath) {
    if (this.platform !== 'win32' && fs.existsSync(binaryPath)) {
      try {
        fs.chmodSync(binaryPath, '755');
      } catch (error) {
        console.warn(`Warning: Could not make binary executable: ${error.message}`);
      }
    }
  }

  /**
   * Setup binary for current platform
   * @param {string} binDir - Directory containing binaries
   * @returns {string} Path to setup binary
   */
  setupBinary(binDir) {
    if (!this.isSupported()) {
      throw new Error(
        `Platform ${this.platform}-${this.arch} is not supported. ` +
        `Supported platforms: macOS (Intel/Apple Silicon), Linux (x64/ARM64), Windows (x64)`
      );
    }

    const binaryPath = this.getBinaryPath(binDir);
    
    if (!fs.existsSync(binaryPath)) {
      const availableBinaries = this.listAvailableBinaries(binDir);
      throw new Error(
        `Binary not found for ${this.platform}-${this.arch}. ` +
        `Expected: ${this.getBinaryName()}. ` +
        `Available binaries: ${availableBinaries.map(b => b.name).join(', ') || 'none'}`
      );
    }

    // Make binary executable on Unix-like systems
    this.makeExecutable(binaryPath);

    return binaryPath;
  }
}

// Export for use as module
module.exports = PlatformDetector;

// CLI usage
if (require.main === module) {
  const detector = new PlatformDetector();
  
  const command = process.argv[2];
  const binDir = process.argv[3] || path.join(__dirname, '..', 'bin');

  switch (command) {
    case 'info':
      console.log(JSON.stringify(detector.getPlatformInfo(), null, 2));
      break;
      
    case 'binary-name':
      console.log(detector.getBinaryName());
      break;
      
    case 'binary-path':
      console.log(detector.getBinaryPath(binDir));
      break;
      
    case 'exists':
      console.log(detector.binaryExists(binDir));
      break;
      
    case 'list':
      const binaries = detector.listAvailableBinaries(binDir);
      console.log(JSON.stringify(binaries, null, 2));
      break;
      
    case 'setup':
      try {
        const binaryPath = detector.setupBinary(binDir);
        console.log(`Binary setup successfully: ${binaryPath}`);
      } catch (error) {
        console.error(`Setup failed: ${error.message}`);
        process.exit(1);
      }
      break;
      
    case 'supported':
      if (detector.isSupported()) {
        console.log('Platform is supported');
        process.exit(0);
      } else {
        console.log('Platform is not supported');
        process.exit(1);
      }
      break;
      
    default:
      console.log(`
Usage: node platform-detector.js <command> [bin-dir]

Commands:
  info        Show platform information
  binary-name Show binary name for current platform
  binary-path Show binary path for current platform
  exists      Check if binary exists for current platform
  list        List all available binaries
  setup       Setup binary for current platform
  supported   Check if current platform is supported

Examples:
  node platform-detector.js info
  node platform-detector.js setup ./bin
  node platform-detector.js exists ./bin
      `);
      break;
  }
}