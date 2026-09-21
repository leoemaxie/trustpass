# TrustPass Deployment Guide

This document details the production deployment process for the TrustPass identity platform. It includes a comprehensive, real-world reference deployment on **Oracle Cloud Infrastructure (OCI)** using Docker Compose, reverse-proxy routing, and SSL termination.

---

## 1. System Requirements & Architecture Overview

TrustPass runs as a containerized stack consisting of 10 microservices, workers, and databases orchestrated via Docker Compose:

| Component | Port | Description |
|---|---|---|
| **Verifier PWA** | `3000` | Shop-owner scan UI (HTML5/Vanilla JS) |
| **Holder Wallet** | `5173` | SvelteKit holder wallet & proof generator |
| **Issuer Console** | `5174` | SvelteKit credential issuance & management UI |
| **Schema Registry** | `8081` | Go schema registry API |
| **Issuer API** | `8082` | Go credential issuance API |
| **Verifier API** | `8083` | Go verification session & replay engine |
| **Receipt Service** | `8084` | Go non-personal audit receipt service |
| **Cryptographic Core** | `50051` | Rust BBS+ & Noir ZK circuit server (internal only) |
| **PostgreSQL 15** | `5432` | Relational database (internal only) |
| **Redis 7** | `6379` | Session TTL & fast cache (internal only) |

### Minimum Hardware Specifications
- **Operating System:** Ubuntu 22.04 LTS or 24.04 LTS (x86_64 or ARM64)
- **CPU:** 2 vCPU / OCPU
- **Memory:** 4 GB RAM minimum (8 GB+ recommended for Rust & Node builds)
- **Disk:** 25 GB SSD storage
- **Container Runtime:** Docker Engine 24.0+ with Docker Compose v2.20+

---

## 2. Configuration & Secret Management

TrustPass strictly follows 12-factor application security:
- **Never commit `.env` or private keys to source control.**
- Copy the template file [`.env.example`](file:///.env.example) to `.env` on your target server.

```bash
cp .env.example .env
```

### Environment Variables Reference
```ini
# Core Cryptographic Service
PORT_CORE=50051
DEFAULT_ISSUER_SEED=13374242   # Replace with a unique random integer in production

# Service Inter-communication URLs
CORE_API_URL=http://core:50051
SCHEMA_REGISTRY_URL=http://schema-registry:8081
ISSUER_API_URL=http://issuer-api:8082
VERIFIER_API_URL=http://verifier-api:8083
RECEIPT_SERVICE_URL=http://receipt-service:8084

# Verification Session Security
VERIFICATION_SESSION_TTL_SECONDS=120

# Database Credentials
POSTGRES_USER=postgres
POSTGRES_PASSWORD=replace_with_a_secure_password
POSTGRES_DB=trustpass
POSTGRES_PORT=5432
REDIS_PORT=6379

# Default Issuer DID
DEFAULT_ISSUER_DID=did:key:zUC724vsrMwHvKyqDdHtrh7z2GNe5xbsfgivth466P4vm2iaJLW9kK48DbgKa32yL944yK9k
```

---

## 3. OCI (Oracle Cloud Infrastructure) Reference Deployment

This section guides you through deploying TrustPass on an Oracle Cloud Compute instance (such as the Always Free tier Ampere A1 or AMD compute VM).

### Step 3.1: Provision the OCI Compute Instance
1. In the **Oracle Cloud Console**, navigate to **Compute** → **Instances** → **Create Instance**.
2. Select **Image:** `Canonical Ubuntu 22.04` (or `Ubuntu 24.04`).
3. Select **Shape:** `VM.Standard.A1.Flex` (Ampere ARM, 2–4 OCPUs, 12–24 GB RAM) or `VM.Standard.E2.1.Micro`.
4. In **Networking**, assign a **Public IPv4 address**.
5. Under **Add SSH keys**, choose **Generate a key pair for me** and download both the **Private Key** (`.key`) and Public Key.
6. Click **Create** and wait for the instance status to show **Running**.
7. Note down your instance's **Public IP Address** (e.g. `<ORACLE_PUBLIC_IP>`).

> [!TIP]
> In Oracle Cloud, navigate to **Compute** → **Instances** → your instance → **IP Addresses** and click **Promote to Reserved Public IP**. This guarantees your IP will never change if the VM is rebooted.

---

### Step 3.2: Configure OCI Virtual Cloud Network (VCN) Ingress Rules
Oracle Cloud filters all inbound traffic at the cloud gateway. You must explicitly allow ingress traffic:

1. In OCI Console, go to **Networking** → **Virtual Cloud Networks (VCN)**.
2. Click your VCN name → select **Security Lists** on the left menu.
3. Click **Default Security List for [your-vcn]** → click **Add Ingress Rules**.
4. Configure the rule:
   - **Source Type:** `CIDR`
   - **Source CIDR:** `0.0.0.0/0` (public access)
   - **IP Protocol:** `TCP`
   - **Source Port Range:** *(leave empty)*
   - **Destination Port Range:** `80,443,3000,5173,5174,8081-8084`
   - **Description:** `TrustPass HTTP, HTTPS, apps, and APIs`
5. Click **Add Ingress Rules**.

---

### Step 3.3: Secure Local SSH Key & Setup Connection Alias (Windows / macOS / Linux)

Keep your downloaded private key in a secure location (never delete it, or you will be locked out):

#### On Windows (PowerShell):
```powershell
# 1. Create .ssh folder and move key to permanent location
mkdir -Force "$HOME\.ssh"
Move-Item "$HOME\Downloads\ssh-key.key" "$HOME\.ssh\oracle_trustpass.key"

# 2. Lock down file permissions (Windows OpenSSH requirement)
icacls "$HOME\.ssh\oracle_trustpass.key" /inheritance:r /grant:r "$($env:USERNAME):(R)"

# 3. Add an SSH host alias to ~/.ssh/config
Set-Content -Path "$HOME\.ssh\config" -Value "Host trustpass", "    HostName <ORACLE_PUBLIC_IP>", "    User ubuntu", "    IdentityFile ~/.ssh/oracle_trustpass.key"
```

#### On Linux / macOS:
```bash
mkdir -p ~/.ssh
mv ~/Downloads/ssh-key.key ~/.ssh/oracle_trustpass.key
chmod 400 ~/.ssh/oracle_trustpass.key

cat <<EOF >> ~/.ssh/config
Host trustpass
    HostName <ORACLE_PUBLIC_IP>
    User ubuntu
    IdentityFile ~/.ssh/oracle_trustpass.key
EOF
```

Connect to your server simply with:
```bash
ssh trustpass
```

---

### Step 3.4: Configure VM Host OS Firewall
Oracle Linux/Ubuntu images include host-level `iptables` rules that block incoming ports by default.

Run these commands inside your SSH session on the VM:
```bash
# Allow inbound traffic on TrustPass application ports
sudo iptables -I INPUT 6 -m state --state NEW -p tcp -m multiport --dports 80,443,3000,5173,5174,8081,8082,8083,8084 -j ACCEPT

# Persist firewall rules across reboots
sudo apt-get update && sudo apt-get install -y iptables-persistent
sudo netfilter-persistent save
```

---

### Step 3.5: Install Docker & Docker Compose
Install the official Docker Engine and Compose plugin:

```bash
# Install prerequisites
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg lsb-release

# Add Docker's official GPG key
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# Add Docker repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker packages
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Enable non-root docker execution
sudo usermod -aG docker $USER
newgrp docker
```

---

### Step 3.6: Clone and Launch the Application Stack

```bash
# Clone repository
git clone https://github.com/leoemaxie/trustpass.git
cd trustpass

# Configure environment
cp .env.example .env
# Edit .env to set your production secrets if necessary:
# nano .env

# Build and launch all 10 containers in the background
docker compose up -d --build
```

Verify service status:
```bash
docker compose ps
```
All containers should report `Up` with `(healthy)` status.

---

## 4. Domain Setup & HTTPS (Mandatory for Phone Camera Access)

> [!IMPORTANT]
> **Why HTTPS is Required:**  
> Modern smartphone browsers (iOS Safari, Android Chrome) block camera access (`navigator.mediaDevices.getUserMedia`) over unencrypted `http://<IP_ADDRESS>`. To use the **Verifier PWA camera scanner** on physical mobile devices, you must terminate traffic with HTTPS.

### Step 4.1: DNS Configuration
At your domain registrar (Cloudflare, Namecheap, GoDaddy, etc.), create `A` records pointing to your `<ORACLE_PUBLIC_IP>`:

| Record Type | Host / Name | Target | Purpose |
|---|---|---|---|
| `A` | `verify` (or `@`) | `<ORACLE_PUBLIC_IP>` | Verifier PWA (`verify.yourdomain.com`) |
| `A` | `wallet` | `<ORACLE_PUBLIC_IP>` | Holder Wallet (`wallet.yourdomain.com`) |
| `A` | `issuer` | `<ORACLE_PUBLIC_IP>` | Issuer Console (`issuer.yourdomain.com`) |

---

### Step 4.2: Reverse Proxy with Automated SSL (Using Caddy)
**Caddy** automatically provisions and renews free Let's Encrypt SSL certificates with zero manual intervention.

On the Oracle VM:
```bash
# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install -y caddy
```

Edit `/etc/caddy/Caddyfile`:
```bash
sudo nano /etc/caddy/Caddyfile
```

Paste this configuration (replacing `yourdomain.com` with your actual domain):
```caddy
# Verifier PWA (Camera Scan Interface)
verify.yourdomain.com {
    reverse_proxy localhost:3000
}

# Holder Wallet (Credential storage & proof generation)
wallet.yourdomain.com {
    reverse_proxy localhost:5173
}

# Issuer Console (Admin dashboard)
issuer.yourdomain.com {
    reverse_proxy localhost:5174
}

# Backend API proxy (optional, for unified domain access)
api.yourdomain.com {
    handle_path /verifier/* {
        reverse_proxy localhost:8083
    }
    handle_path /issuer/* {
        reverse_proxy localhost:8082
    }
    handle_path /schemas/* {
        reverse_proxy localhost:8081
    }
    handle_path /receipts/* {
        reverse_proxy localhost:8084
    }
}
```

Reload Caddy:
```bash
sudo systemctl reload caddy
```
Caddy will automatically request and install SSL certificates for all specified subdomains within seconds.

---

### Alternative: Instant HTTPS via Cloudflare Tunnel (No Domain Required)
If you do not own a custom domain, use Cloudflare's free quick tunnel for instant HTTPS:

```bash
# Install cloudflared on the VM
curl -L -o cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb

# Start tunnel pointing to Verifier PWA
cloudflared tunnel --url http://localhost:3000
```
`cloudflared` outputs an instant URL (e.g. `https://random-words.trycloudflare.com`). Open this link on your smartphone to scan QR codes using the device camera.

---

## 5. Operations & Maintenance

### Viewing Live Service Logs
```bash
# View all container logs
docker compose logs -f

# View logs for a specific service
docker compose logs -f verifier-api
docker compose logs -f core
```

### Restarting or Updating Services
```bash
# Pull latest repository changes
git pull origin main

# Rebuild and reload affected containers with zero downtime
docker compose up -d --build
```

### Backing Up Data
PostgreSQL data is stored in the Docker volume `postgres_data`. To perform a logical backup:
```bash
docker exec -t trustpass-postgres pg_dumpall -c -U postgres > backup_$(date +%Y%m%d).sql
```
