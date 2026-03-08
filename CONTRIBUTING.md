# Contributing

Thanks for your interest in contributing to SREKubeCraft Demo!

## How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b my-demo`)
3. Add your demo in a new directory following the existing structure
4. Include a `README.md` with prerequisites, quick start, and demo instructions
5. Use [Taskfile](https://taskfile.dev/) for automation
6. Commit your changes and open a pull request

## Demo Structure

Each demo directory should contain:

- `README.md` with architecture overview, prerequisites, and step-by-step instructions
- `Taskfile.yml` for setup, demo, and cleanup automation
- `app/` for any application code (Dockerfile included)
- `kubernetes/` for manifests organized by component

## Guidelines

- Keep demos self-contained and reproducible on a local Kind cluster
- Use pinned versions for all dependencies
- Include cleanup tasks to remove all resources
- Test the full setup-to-demo flow before submitting

## Reporting Issues

Open an issue at [github.com/nicknikolakakis/srekubecraft-demo/issues](https://github.com/nicknikolakakis/srekubecraft-demo/issues).

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
