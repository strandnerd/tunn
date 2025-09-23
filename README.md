# tunn - SSH Tunnel Manager

`tunn` is a developer-friendly wrapper around OpenSSH that makes it easy to manage multiple SSH tunnels defined in a simple configuration file

## Features

- 🚀 **Simple Configuration**: Define all your tunnels in a single YAML file
- 🔧 **Selective Tunnels**: Run all tunnels or specific ones by name
- 🔌 **Multiple Ports**: Support for multiple port mappings per tunnel
- 🔐 **SSH Integration**: Leverages your existing SSH configuration
- ⚡ **Parallel Execution**: All tunnels run concurrently

## Installation

### From Source

```bash
go install github.com/strandnerd/tunn@latest
```

### Build Locally

```bash
git clone https://github.com/strandnerd/tunn.git
cd tunn
go build -o tunn
sudo mv tunn /usr/local/bin/
```

## Configuration

Create a `~/.tunnrc` file in your home directory:

```yaml
tunnels:
  api:
    host: myserver          # SSH host from ~/.ssh/config
    ports:
      - 3000:3000           # local:remote port mapping
      - 4000:4001
    user: apiuser           # optional: SSH user
    identity_file: ~/.ssh/id_rsa  # optional: SSH key

  db:
    host: database
    ports:
      - 3306:3306           # MySQL
      - 5432:5432           # PostgreSQL
    user: dbadmin           # optional: overrides SSH config

  cache:
    host: cacheserver
    ports:
      - 6379:6379           # Redis
```

### Configuration Fields

- `tunnels`: Map of tunnel names
- `host`: SSH host alias from `~/.ssh/config`
- `ports`: List of port mappings in `local:remote` format
- `user` (optional): SSH username (overrides `~/.ssh/config`)
- `identity_file` (optional): Path to SSH private key

## Usage

### Run All Tunnels

```bash
tunn
```

### Run Specific Tunnels

```bash
# Single tunnel
tunn api

# Multiple tunnels
tunn api db

# All database-related tunnels
tunn db cache
```

### Output Example

```
Tunnels Ready

[api]
    3000 ➜ 3000 [active]
    4000 ➜ 4001 [active]
[db]
    3306 ➜ 3306 [connecting]
    5432 ➜ 5432 [active]
```

## SSH Configuration

`tunn` uses your system's SSH configuration. Make sure your hosts are defined in `~/.ssh/config`:

```ssh
Host myserver
    HostName 192.168.1.100
    User myuser
    Port 22

Host database
    HostName db.example.com
    User dbuser
    IdentityFile ~/.ssh/db_key
```

## Requirements

- Go 1.21 or higher (for building)
- OpenSSH client (`ssh` command)
- Valid SSH configuration
