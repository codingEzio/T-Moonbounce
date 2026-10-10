# Contributing

Keep the driving screen minimal and the game playable offline. Prefer focused changes and explain their visible effect.

- Run `go test ./...`, `go vet ./...` and `gofmt` on changed Go files.
- For UI or handling changes, play the app at 980×680 and a larger size. Check English and both Chinese translations, Pixel/System fonts, and light/dark appearance where affected.
- For audio changes, use the muted transport check in the README before an intentional listening check.
- Use `--state .local/example.json` for isolated tests. Never commit real saves, credentials, recordings or machine-specific paths.
- Include source, version and license information for any new dependency or asset.

Report reproducible bugs through GitHub Issues with the app/Go/macOS version and concise steps. Review screenshots and logs for private information before posting. Do not publish credentials or personal data in an issue.

Contributions to project-owned code and art are submitted under the root MIT license. Third-party assets retain their own licenses and notices.
