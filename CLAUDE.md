# Hermes Peer Score Tool

A Go CLI tool for analyzing Ethereum network peer connection health and performance using Hermes as a gossipsub listener for beacon nodes. This tool monitors peer connections, analyzes network behavior, and generates comprehensive reports with optional AI-powered insights.

## Project Structure
Claude MUST read the `.cursor/rules/project_architecture.mdc` file before making any structural changes to the project.

## Code Standards  
Claude MUST read the `.cursor/rules/code_standards.mdc` file before writing any code in this project.

## Development Workflow
Claude MUST read the `.cursor/rules/development_workflow.mdc` file before making changes to build, test, or deployment configurations.

## Component Documentation
Individual components have their own CLAUDE.md files with component-specific rules. Always check for and read component-level documentation when working on specific parts of the codebase.

## Key Validation Modes
This project supports two distinct validation modes that require different Hermes versions:
- **Delegated Validation**: Uses Hermes v0.0.4-0.20250513093811-320c1c3ee6e2, delegates to Prysm
- **Independent Validation**: Uses Hermes v0.0.4-0.20250611164742-0abea7d82cb4, internal validation

Use the built-in commands to switch between modes:
```bash
./peer-score-tool --validation-mode=delegated --update-go-mod
./peer-score-tool --validation-mode=independent --update-go-mod
```

## Testing and Quality
- Always test both validation modes when making changes
- Use `go fmt`, `go vet`, and run tests before committing
- CI workflows validate both modes with real network conditions
- Follow structured logging practices with logrus throughout the codebase