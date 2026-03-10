# Clauductor 🚂

**Your AI needs a conductor**

A beautiful web interface for Claude Code with real-time work visualization, session management, and project tracking.

## Features

- 🗺️ **Real-time Work Map** - Visualize Claude's work with an interactive graph
- 💬 **Beautiful Chat Interface** - Modern UI with syntax highlighting and streaming
- 📁 **Project Management** - Organize work with projects and session history  
- 🔐 **Permission Management** - Interactive prompts with YOLO mode
- ⚙️ **Profiles & Settings** - Configure API keys and environment profiles
- 🔄 **Auto-Updates** - Built-in update system with GitHub release notes

## Quick Install

### Linux / macOS

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/mikolajbadyl/clauductor@main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://cdn.jsdelivr.net/gh/mikolajbadyl/clauductor@main/install.ps1 | iex
```

**No sudo required** • Installs to `~/.local/bin` • Automatic PATH setup

## Usage

After installation:

```bash
clauductor              # Start the server
clauductor --help       # Show all options
```

Then open http://localhost:3420 in your browser.

## Documentation

Visit our [landing page](https://mikolajbadyl.github.io/clauductor-landingpage) for more information.

## Development

```bash
# Install dependencies
npm install

# Build frontend
make frontend

# Build full application
make build

# Run in dev mode
make dev
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
