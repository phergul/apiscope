# apiscope

`apiscope` is a terminal API explorer for Swagger 2.0 and OpenAPI 3.x specs.

## Features

- Load API specs from a local file path or URL.
- Parse and normalize Swagger/OpenAPI specs into one internal model.
- Browse operations, parameters, request bodies, and responses in the schema explorer.
- Execute API operations and inspect response status, headers, body, and timing.
- Keep request history so you can revisit earlier calls per spec.
- Reload specs to refresh operations without restarting the app.

## Install

```bash
go install github.com/phergul/apiscope@latest
```

## Usage

```bash
apiscope <spec-source>
```
