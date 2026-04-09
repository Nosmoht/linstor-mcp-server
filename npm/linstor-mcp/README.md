# linstor-mcp

MCP server for LINSTOR and the Piraeus operator with staged plan/apply safety.

Connects AI agents to LINSTOR inventory, planning, and controlled cluster-configuration apply flows without scraping shell output.

## Installation

```bash
npx linstor-mcp
```

No Go toolchain required. The correct native binary for your platform (Linux/macOS, x64/ARM64) is installed automatically.

## Configuration

Use the same environment variables documented in the main repository README, including `LINSTOR_MCP_CONFIG`, `LINSTOR_MCP_PROFILE`, and the `LINSTOR_MCP_*` controller overrides.

## Documentation

Full docs, client setup, tool reference, and security model:

**[https://github.com/Nosmoht/linstor-mcp-server](https://github.com/Nosmoht/linstor-mcp-server)**

## License

[MIT](https://github.com/Nosmoht/linstor-mcp-server/blob/main/LICENSE)
