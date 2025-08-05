# CCDash Nginx Setup - Template Configuration

Simple nginx proxy configuration to access CCDash.

## File Structure

- `ccdash.conf.template` - nginx configuration template (managed by Git)
- `ccdash.conf` - personal nginx configuration (auto-generated, gitignored)
- `setup.sh` - setup script that displays manual steps
- `README.md` - this document

## Template Approach

- **`ccdash.conf.template`**: Common base configuration managed by Git
- **`ccdash.conf`**: Personal configuration generated from template, gitignored and freely customizable

## Setup Steps

### 1. Run Setup Script

```bash
cd nginx
./setup.sh
```

This script will:
- Auto-generate personal configuration `ccdash.conf` from template (first time only)
- Display manual commands to execute

### 2. Manually Install nginx Configuration

#### For macOS (Homebrew):
```bash
sudo cp ccdash.conf /opt/homebrew/etc/nginx/servers/
sudo sed -i '' 's/listen.*8080/listen       8888/' /opt/homebrew/etc/nginx/nginx.conf
sudo nginx -t
sudo nginx -s reload
```

#### For Ubuntu/Debian:
```bash
sudo cp ccdash.conf /etc/nginx/sites-available/
sudo ln -sf /etc/nginx/sites-available/ccdash.conf /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo nginx -s reload
```

### 3. Start CCDash Services

#### Backend (separate terminal):
```bash
cd ../backend
go run cmd/server/main.go
```

#### Frontend (separate terminal):
```bash
cd ../frontend
npm run dev
```

### 4. Access

Access CCDash at http://localhost

## Configuration Details

### nginx Configuration (`ccdash.conf`)

```nginx
server {
    listen 80;
    server_name localhost;

    # Frontend proxy to Next.js dev server
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # API proxy to backend
    location /api/ {
        proxy_pass http://127.0.0.1:6060/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Proxy Configuration

- **Frontend**: `http://localhost/` → `http://127.0.0.1:3000/`
- **API**: `http://localhost/api/` → `http://127.0.0.1:6060/api/`

## Troubleshooting

### Common Issues

#### 1. Port 80 already in use
```bash
# Check which process is using the port
sudo lsof -i :80

# Stop other services if necessary
```

#### 2. nginx configuration errors
```bash
# Test configuration
sudo nginx -t

# Check error logs
sudo tail -f /var/log/nginx/error.log  # Linux
sudo tail -f /opt/homebrew/var/log/nginx/error.log  # macOS
```

#### 3. Services not running
```bash
# Check processes
ps aux | grep "go run\|next-server\|nginx"

# Start each service individually
```

### Removing Configuration

To remove the configuration:

#### macOS:
```bash
sudo rm /opt/homebrew/etc/nginx/servers/ccdash.conf
sudo sed -i '' 's/listen.*8888/listen       8080/' /opt/homebrew/etc/nginx/nginx.conf
sudo nginx -s reload
```

#### Ubuntu/Debian:
```bash
sudo rm /etc/nginx/sites-enabled/ccdash.conf
sudo rm /etc/nginx/sites-available/ccdash.conf
sudo nginx -s reload
```

## Verification

To verify the configuration is working correctly:

1. **Test nginx configuration**: `sudo nginx -t`
2. **Check service status**: `./setup.sh`
3. **Access via browser**: http://localhost
4. **Test API**: http://localhost/api/v1/health

## Customization

You can freely edit the personal configuration `ccdash.conf`:

```bash
# Customize configuration
vi nginx/ccdash.conf

# Example: Change server_name
server_name example.com;

# Example: Use different port
listen 8080;
```

After changes, reload nginx:
```bash
sudo nginx -s reload
```

## Template Updates

To update common configuration:

1. Edit `ccdash.conf.template`
2. Remove existing personal config and regenerate:
   ```bash
   rm ccdash.conf
   ./setup.sh
   ```

## CORS and Private IP Support

### Automatic Private IP Support (v0.5.7+)

The backend now automatically allows CORS requests from **any private IP address**, making it work seamlessly with nginx on different network configurations:

**Supported IP ranges:**
- `10.0.0.0/8` (Class A private)
- `172.16.0.0/12` (Class B private) 
- `192.168.0.0/16` (Class C private)
- `127.0.0.0/8` (Loopback)

**Examples of automatically allowed origins:**
- `http://192.168.1.100` 
- `http://10.0.0.50`
- `http://172.16.0.10`
- `http://localhost`, `http://127.0.0.1`

### Security Features

- ✅ Only HTTP/HTTPS protocols allowed
- ✅ Only standard ports (80, 443, 3000, 8080) allowed for private IPs
- ✅ Public IP addresses are **not** automatically allowed
- ✅ Explicit origin allow-list still supported via `CORS_ALLOWED_ORIGINS`

### Configuration Options

```bash
# Allow specific additional origins
export CORS_ALLOWED_ORIGINS="https://mydomain.com,http://custom-server:8080"

# Allow all origins (development only - not recommended)
export CORS_ALLOW_ALL=true

# Then start the backend
cd backend && go run cmd/server/main.go
```

### Why This Works

1. **nginx proxy**: Requests from `http://192.168.3.5/api/` → `http://localhost:6060/api/`
2. **Browser origin**: Sees request as coming from `http://192.168.3.5`
3. **Backend CORS**: Automatically allows private IP `192.168.3.5`
4. **No CORS errors**: ✅ Cross-origin requests work without manual configuration

This eliminates the need to manually configure CORS for each user's specific IP address while maintaining security by only allowing private network access.

## HTTPS Setup for Production

### SSL Certificate Setup

#### Option 1: Let's Encrypt (Recommended)

```bash
# Install Certbot
sudo apt update && sudo apt install certbot python3-certbot-nginx

# Obtain SSL certificate
sudo certbot certonly --nginx -d ccdash.example.com

# Certificates will be installed at:
# /etc/letsencrypt/live/ccdash.example.com/fullchain.pem
# /etc/letsencrypt/live/ccdash.example.com/privkey.pem
```

#### Option 2: Self-signed Certificate (Development/Testing)

```bash
# Generate self-signed certificate
sudo mkdir -p /etc/nginx/ssl
sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/nginx/ssl/ccdash.key \
  -out /etc/nginx/ssl/ccdash.crt \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=ccdash.local"
```

### HTTPS Configuration

#### 1. Create HTTPS Configuration

```bash
# Copy HTTPS template
cp ccdash-https.conf.template ccdash-https.conf

# Edit for your domain
sed -i 's/ccdash.example.com/your-domain.com/g' ccdash-https.conf

# For self-signed certificates, update certificate paths:
# ssl_certificate /etc/nginx/ssl/ccdash.crt;
# ssl_certificate_key /etc/nginx/ssl/ccdash.key;
```

#### 2. Install HTTPS Configuration

##### Ubuntu/Debian:
```bash
sudo cp ccdash-https.conf /etc/nginx/sites-available/
sudo ln -sf /etc/nginx/sites-available/ccdash-https.conf /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo nginx -s reload
```

##### macOS (Homebrew):
```bash
sudo cp ccdash-https.conf /opt/homebrew/etc/nginx/servers/
sudo nginx -t
sudo nginx -s reload
```

#### 3. Configure Environment Variables

```bash
# Backend (.env)
GIN_MODE=release
CCDASH_API_KEY=your-secure-api-key-here
CORS_ALLOWED_ORIGINS=https://your-domain.com

# Frontend (.env.local)
NEXT_PUBLIC_API_URL=https://your-domain.com/api
NEXT_PUBLIC_API_KEY=your-secure-api-key-here
```

#### 4. Security Features Included

✅ **HTTP to HTTPS redirect**  
✅ **Strong SSL/TLS configuration**  
✅ **Security headers (HSTS, CSP, etc.)**  
✅ **OCSP Stapling**  
✅ **Gzip compression**  
✅ **Rate limiting for API endpoints**  
✅ **Static asset caching**  
✅ **Optional nginx-level API key validation**  

### Certificate Auto-renewal

For Let's Encrypt certificates:

```bash
# Test renewal (dry run)
sudo certbot renew --dry-run

# Set up automatic renewal (crontab)
sudo crontab -e
# Add this line:
0 12 * * * /usr/bin/certbot renew --quiet
```

### Firewall Configuration

```bash
# Allow HTTPS traffic
sudo ufw allow 443/tcp
sudo ufw allow 80/tcp  # For Let's Encrypt challenges and redirects
sudo ufw enable
```

### Production Checklist

- [ ] SSL certificate installed and valid
- [ ] HTTP to HTTPS redirect working
- [ ] All security headers present
- [ ] API Key authentication enabled (`GIN_MODE=release`)
- [ ] CORS properly configured
- [ ] Firewall configured
- [ ] Certificate auto-renewal set up
- [ ] Rate limiting configured
- [ ] Error pages customized

### Testing HTTPS Setup

```bash
# Test SSL configuration
curl -I https://your-domain.com

# Test API with authentication
curl -H "X-API-Key: your-api-key" https://your-domain.com/api/health

# Check SSL rating
# Visit: https://www.ssllabs.com/ssltest/analyze.html?d=your-domain.com
```

## Important Notes

### Development vs Production

- **Development**: Use `ccdash.conf.template` (HTTP only, permissive CORS)
- **Production**: Use `ccdash-https.conf.template` (HTTPS, strict security)

### Configuration Files

- `ccdash.conf.template` - Development HTTP configuration
- `ccdash-https.conf.template` - Production HTTPS configuration  
- `ccdash.conf` / `ccdash-https.conf` - Personal configurations (gitignored)

### Security Notes

- Both frontend and backend services must be running
- API Key authentication is required in production (`GIN_MODE=release`)
- Private IP CORS support is automatic for development
- HTTPS is mandatory for production deployments
- Regular security updates recommended