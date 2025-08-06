// Frontend Authentication Helper
// This approach keeps API keys secure on the server side

interface AuthResponse {
  token: string;
  expires: number;
}

class FrontendAuth {
  private token: string | null = null;
  private tokenExpiry: number = 0;

  // Authenticate with backend using a session-based approach
  async authenticate(password?: string): Promise<boolean> {
    try {
      // Option 1: Simple password-based auth (development)
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password: password || 'dev-password' })
      });

      if (response.ok) {
        const data: AuthResponse = await response.json();
        this.token = data.token;
        this.tokenExpiry = data.expires;
        return true;
      }
      return false;
    } catch (error) {
      console.error('Authentication failed:', error);
      return false;
    }
  }

  // Get current session token
  getToken(): string | null {
    if (this.token && Date.now() < this.tokenExpiry) {
      return this.token;
    }
    return null;
  }

  // Check if authenticated
  isAuthenticated(): boolean {
    return this.getToken() !== null;
  }

  // Logout
  logout(): void {
    this.token = null;
    this.tokenExpiry = 0;
  }
}

export const frontendAuth = new FrontendAuth();