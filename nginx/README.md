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

## Important Notes

- This configuration is for development use
- Production environments require additional security settings
- SSL/HTTPS configuration is not included
- Both frontend and backend services must be running
- `ccdash.conf` is gitignored, so configurations are not shared between team members