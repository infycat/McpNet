# Plugin examples

This directory is the only plugin content intended for the public repository. It contains a deliberately minimal, disabled, read-only starter; it is not a collection of personal protocol plugins.

The application loads **user-installed plugins** from `%USERPROFILE%\McpNet_new\plugins\<plugin-id>\`, not from this repository directory. To try the example, copy its folder into that user-data directory, review `main.go`, then explicitly enable it in the application.

Do not commit personal finished plugins (including protocol-specific XXTEA/XOR implementations), keys, captures, or user configuration. The repository `.gitignore` allows only this examples directory under `/plugins/`.

The runtime uses Yaegi and offers the `mcpnet` package helpers documented in the project architecture guide. Hooks include `OnHTTPRequest`, `OnHTTPResponse`, `OnWebSocket`, `OnUDP`, and `OnURLDisplay`. A plugin can change payloads through `SetBody` and display-only URLs through `SetURL`; only install code you trust.
