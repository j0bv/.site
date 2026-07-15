# Setup Guide

## Quick Start

1. **Install Go** (1.21+)
   ```bash
   # On Ubuntu/Debian
   sudo apt install golang-go
   
   # On macOS
   brew install go
   
   # Verify installation
   go version
   ```

2. **Install PostgreSQL**
   ```bash
   # On Ubuntu/Debian
   sudo apt install postgresql postgresql-contrib
   
   # Create database
   sudo -u postgres createdb truck_inventory
   ```

3. **Clone and Setup**
   ```bash
   cd truck-inventory
   make deps
   make setup-env
   # Edit .env with your credentials
   ```

4. **Build and Run**
   ```bash
   make build
   ./truck-inventory
   ```

## Detailed Setup

### 1. Database Configuration

The application will automatically create the database schema on first run. Ensure PostgreSQL is running and accessible:

```bash
# Start PostgreSQL
sudo systemctl start postgresql

# Create database user (optional)
sudo -u postgres createuser truck_user
sudo -u postgres psql -c "ALTER USER truck_user WITH PASSWORD 'your_password';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE truck_inventory TO truck_user;"
```

Update `.env` with your database credentials.

### 2. Microsoft Graph API Setup

#### Step 1: Register Application in Azure AD

1. Go to [Azure Portal](https://portal.azure.com)
2. Navigate to **Azure Active Directory** > **App registrations**
3. Click **New registration**
4. Name: `Truck Inventory Service`
5. Supported account types: **Accounts in this organizational directory only**
6. Click **Register**

#### Step 2: Create Client Secret

1. In your app registration, go to **Certificates & secrets**
2. Click **New client secret**
3. Description: `Truck Inventory Secret`
4. Expires: Choose appropriate duration
5. Click **Add**
6. **Copy the secret value immediately** (you won't see it again)

#### Step 3: Configure API Permissions

1. Go to **API permissions**
2. Click **Add a permission**
3. Select **Microsoft Graph**
4. Select **Application permissions**
5. Add:
   - `Mail.Read` (Read mail in all mailboxes)
   - `Mail.Send` (Send mail as any user)
6. Click **Add permissions**
7. Click **Grant admin consent** (requires admin privileges)

#### Step 4: Get Tenant and App IDs

1. From **Overview** page, copy:
   - **Application (client) ID** → `MICROSOFT_APP_ID`
   - **Directory (tenant) ID** → `MICROSOFT_TENANT_ID`
2. Use the client secret from Step 2 → `MICROSOFT_APP_SECRET`

#### Step 5: Determine User ID

Since the application uses **Application permissions** (not delegated), you must specify which user's mailbox to monitor. The user ID can be:
- User Principal Name (UPN): `user@example.com`
- User Object ID: A GUID from Azure AD

To find a user's Object ID:
1. Go to **Azure Active Directory** > **Users**
2. Find the user whose mailbox you want to monitor
3. Copy either the **User principal name** or **Object ID**

#### Step 6: Update .env File

```bash
MICROSOFT_APP_ID=your_app_id_here
MICROSOFT_APP_SECRET=your_secret_here
MICROSOFT_TENANT_ID=your_tenant_id_here
MICROSOFT_USER_ID=user@example.com
```

**Note**: `MICROSOFT_USER_ID` can be either the email address (UPN) or the Object ID of the user whose mailbox should be monitored.

### 3. IronConnect Setup

1. Create an account at [IronConnect](https://ironconnect.com)
2. Add credentials to `.env`:
   ```bash
   IRON_EMAIL=your_email@example.com
   IRON_PASSWORD=your_password
   ```

### 4. Testing extraction

Before running the full scraper, test SLM-based extraction (or the shared fallback when `SLM_ENABLED=false`):

```bash
make test-extraction
```

Runs sample HTML through `agent.ExtractListingFromHTML` and `agent.ExtractListingsFromHTML` to verify the SLM and prompt.

### 5. Production Deployment

#### Using Systemd

```bash
# Copy service file
sudo cp deployment/truck-inventory.service /etc/systemd/system/

# Edit service file to match your paths
sudo nano /etc/systemd/system/truck-inventory.service

# Create service user
sudo useradd -r -s /bin/false -d /opt/truck-inventory truck_service

# Create directory
sudo mkdir -p /opt/truck-inventory
sudo cp truck-inventory /opt/truck-inventory/
sudo cp .env /opt/truck-inventory/
sudo chown -R truck_service:truck_service /opt/truck-inventory

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable truck-inventory
sudo systemctl start truck-inventory

# Check status
sudo systemctl status truck-inventory
sudo journalctl -u truck-inventory -f
```

#### Using Deployment Script

```bash
sudo ./deployment/deploy.sh
```

## Troubleshooting

### Database Connection Issues

```bash
# Test PostgreSQL connection
psql -h localhost -U postgres -d truck_inventory

# Check if PostgreSQL is running
sudo systemctl status postgresql

# Check connection string format
# Should be: host=localhost port=5432 user=postgres password=xxx dbname=truck_inventory
```

### Microsoft Graph Authentication Issues

- Verify all four environment variables are set (MICROSOFT_APP_ID, MICROSOFT_APP_SECRET, MICROSOFT_TENANT_ID, MICROSOFT_USER_ID)
- Check that admin consent was granted for API permissions
- Verify client secret hasn't expired
- Ensure MICROSOFT_USER_ID is correct (can use UPN or Object ID)
- Check Azure AD logs for authentication errors
- Verify the user specified in MICROSOFT_USER_ID exists and has a mailbox

### Scraper Issues

- Run `make test-extraction` to verify SLM or fallback extraction
- Verify IronConnect login credentials (IRON_EMAIL, IRON_PASSWORD)
- Check that SLM is reachable at SLM_URL when SLM_ENABLED is not "false"

### Email Matching Not Working

- Verify emails are being fetched (check logs)
- Test search criteria extraction with sample emails
- Check database for saved email requests
- Verify matching queries are finding trucks

## Next Steps

1. Test extraction: `make test-extraction`
2. ✅ Verify database schema is created
3. ✅ Test Microsoft Graph API connection
4. ✅ Run scraper manually to verify data collection
5. ✅ Send test email and verify matching works
6. ✅ Deploy to production server

## Support

For issues or questions:
- Check logs: `journalctl -u truck-inventory -f`
- Review database: `psql -d truck_inventory`
- Test individual components using test scripts
